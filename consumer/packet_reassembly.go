package consumer

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/kthomas/go-natsutil"
	"github.com/kthomas/go-redisutil"
	"github.com/provideapp/goldmine/common"
)

const packetReassemblyHeaderKeySuffix = "header"
const packetReassemblyFragmentIngestCountKeySuffix = "fragments.ingest-count"
const packetReassemblyFragmentPersistenceKeySuffix = "fragments.persistence"

const packetReassemblyFragmentationChunkSize = float64(4500)

// Fragmentable interface
type Fragmentable interface {
	Broadcast() error
	Verify() (bool, error)
}

// BroadcastFragments splits the given packet into chunks, optionally padding the last chunk (i.e., with null
// bytes such that all fragments are of equal length). Each chunk is broadcast as a fragment for remote packet
// reassembly, the first such `packetFragment` containing a header describing the `packetReassembly`. If a
// `nextHop` subject is provided, a pointer to the reassembled packet is forwarded upon ingestion of all
// fragments such that reassembly of the packet is guaranteed within the `nextHop` handler at runtime (i.e.,
// such a handler can be thought of as an "exit node" for the reassembled packet and terminates execution on it).
func BroadcastFragments(packet []byte, pad bool, nextHop *string) error {
	packetSize := len(packet)
	chunkSize := int(packetReassemblyFragmentationChunkSize)
	common.Log.Debugf("fragmented broadcast of %d-byte packet into %d-byte chunks requested...", packetSize, chunkSize)

	numChunks := 1
	if packetSize > chunkSize {
		numChunks = int(math.Ceil(float64(packetSize) / float64(chunkSize)))
	}

	packetReassembly := packetReassemblyFactory(packet, nextHop)
	fragments := make([]*packetFragment, numChunks)

	i := 0
	n := len(packet) // n stores the number of aggregate bytes remaining

	common.Log.Debugf("chunking %d-byte packet into %d %d-byte chunks", packetSize, numChunks, chunkSize)
	for i <= numChunks-1 {
		x := i * chunkSize
		y := x + chunkSize - 1

		common.Log.Debugf("preparing %d-byte chunk %d of %d for fragmented broadcast; slice: (%d, %d)", chunkSize, i+1, numChunks, x, y)

		padsize := 0
		if n < chunkSize {
			if pad {
				padsize = chunkSize - n
				y = x + chunkSize - padsize
			} else {
				y = x + n
			}
		}

		var fragmentPayload []byte

		if padsize == 0 {
			fragmentPayload = packet[x:y]
		} else if padsize > 0 {
			common.Log.Debugf("adding %d null bytes to pad chunk %d of %d to uniform %d-byte chunksize", padsize, i, numChunks, chunkSize)
			fragmentPayload = make([]byte, y-x+padsize)
			copy(fragmentPayload[:], packet[x:y])
		}

		checksum := md5.Sum(fragmentPayload)

		fragments[i] = &packetFragment{
			Index:           uint(i),
			Cardinality:     uint(numChunks),
			Checksum:        common.StringOrNil(string(checksum[:])),
			Nonce:           packetReassembly.Nonce,
			Padded:          padsize > 0,
			Payload:         common.StringOrNil(string(fragmentPayload)),
			PayloadChecksum: packetReassembly.Checksum,
		}

		if i == 0 {
			fragments[i].Reassembly = packetReassembly
		}

		n = n - chunkSize
		i++

		common.Log.Debugf("%d-byte chunk %d of %d prepared for remote reassembly; %d total bytes remaining", chunkSize, i, numChunks, n)
	}

	common.Log.Debugf("Prepared %d packet fragments for broadcast", len(fragments))
	for _, fragment := range fragments {
		err := fragment.Broadcast()
		if err != nil {
			common.Log.Warningf("failed to broadcast fragment %d of %d ; %s", fragment.Index, numChunks, err.Error())
			return err
		}
	}

	return nil
}

// packetReassemblyFactory constructs a new packet reassembly instance which can be used
// as a header within the first fragment of a distributed packet reassembly operation
func packetReassemblyFactory(packet []byte, nextHop *string) *packetReassembly {
	checksum := md5.Sum(packet)
	nonce := time.Now().UnixNano()

	return &packetReassembly{
		Cardinality: uint(math.Ceil(math.Mod(float64(len(packet)), packetReassemblyFragmentationChunkSize))),
		Checksum:    common.StringOrNil(string(checksum[:])),
		Next:        nextHop,
		Nonce:       &nonce,
		Size:        uint(len(packet)),
		Payload:     &packet,
	}
}

// packetReassemblyIndexKeyFactory returns a unique identifier for in-memory cache & mutexes,
// fragment persistent storage facilities, etc. for in-flight packet reassembly operations
// for the given Fragmentable
func packetReassemblyIndexKeyFactory(fragmentable Fragmentable, suffix *string) *string {
	var checksum *string
	var nonce *int64

	switch fragmentable.(type) {
	case *packetFragment:
		checksum = fragmentable.(*packetFragment).PayloadChecksum
		nonce = fragmentable.(*packetFragment).Nonce
	case *packetReassembly:
		checksum = fragmentable.(*packetReassembly).Checksum
		nonce = fragmentable.(*packetFragment).Nonce
	default:
		common.Log.Warning("Reflection not supported for given fragmentable")
	}

	if checksum == nil {
		common.Log.Warningf("Failed to resolve checksum of fragmentable during attempt to construct index key")
		return nil
	}

	digest := sha256.New()
	digest.Write([]byte(fmt.Sprintf("%s.%d.%s", natsPacketReassembleSubject, *nonce, *checksum)))
	key := hex.EncodeToString(digest.Sum(nil))

	if suffix != nil {
		key = fmt.Sprintf("%s.%s", key, *suffix)
	}

	return &key
}

// packetFragment represents a packet fragment ingest message payload; TODO: support marshaling from wire/protocol in addition to JSON
type packetFragment struct {
	Index           uint              `json:"index"`                          // i.e., the index of the fragment
	Cardinality     uint              `json:"cardinality"`                    // i.e., # of total fragments comprising the packet
	Checksum        *string           `json:"checksum"`                       // i.e., md5 checksum of the fragment payload
	Nonce           *int64            `json:"nonce"`                          // i.e., nonce associated with the packet reassembly operation
	Padded          bool              `json:"padded"`                         // i.e., true if the fragment payload is padded with null bytes to match a uniform chunksize
	Payload         *string           `json:"payload"`                        // i.e., the raw fragment payload
	PayloadChecksum *string           `json:"reassembled_checksum,omitempty"` // i.e., md5 checksum of the entire n of n payload
	Reassembly      *packetReassembly `json:"reassembly,omitempty"`           // pointer to the packet reassembly header

	// TODO: forward secrecy considerations
}

// Broadcast marshals and transmits the fragment metadata and payload
func (p *packetFragment) Broadcast() error {
	payload, err := json.Marshal(p)
	if err != nil {
		return err
	}
	common.Log.Debugf("attempting to broadcast %d-byte fragment", len(payload))
	return natsutil.NatsStreamingPublish(natsPacketFragmentIngestSubject, payload)
}

// FetchReassemblyHeader fetches the previously-cached packet reassembly header, warms the fragment-local
// `Reassembly` reference and returns the loaded `packetReassembly` -- or returns nil and an error if the
// header failed to load for any reason.
func (p *packetFragment) FetchReassemblyHeader() (*packetReassembly, error) {
	headerKey := packetReassemblyIndexKeyFactory(p, common.StringOrNil(packetReassemblyHeaderKeySuffix))
	if headerKey == nil {
		return nil, errors.New("Failed to fetch cached reassembly header; packet reassembly key factory returned nil header key")
	}

	rawval, err := redisutil.Get(*headerKey)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch cached reassembly header for ; %s", err.Error())
	} else if rawval == nil {
		return nil, fmt.Errorf("Failed to parse valid reassembly header; received nil value from cache")
	}

	var reassembly *packetReassembly
	err = json.Unmarshal([]byte(*rawval), &reassembly)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal cached reassembly header; %s", err.Error())
	}

	return reassembly, nil
}

// Ingest verifies the checksum of the fragment, writes the the underlying bytes to persistent
// or ephemeral storage, atomically increments the internal ingest counter associated with the
// packet reassembly operation and returns the boolean checksum verification result, total number
// of fragments ingested (i.e., after ingesting this fragment-- or nil, if ingestion failed) and
// any error if the attempt to ingest the fragment fails
func (p *packetFragment) Ingest() (bool, *uint, error) {
	if p.Checksum == nil {
		return false, nil, errors.New("no fragment ingestion attempted; nil checksum")
	}

	if len(*p.Checksum) != 16 { // require 128-bit checksum
		return false, nil, fmt.Errorf("no fragment ingestion attempted; 128-bit checksum required for verification (%d-bit checksum found: %s)", len(*p.Checksum)*8, *p.Checksum)
	}

	if p.Payload == nil {
		return false, nil, errors.New("failed to ingest fragment; nil payload")
	}

	verified, err := p.Verify()
	if err != nil || !verified {
		msg := fmt.Sprintf("fragment ingest or checksum verification failed")
		if err != nil {
			msg = fmt.Sprintf("%s; %s", msg, err.Error())
		}
		return false, nil, errors.New(msg)
	}

	common.Log.Debugf("incomplete Ingest() implementation... TODO: persist %d-byte fragment", len(*p.Payload))

	ingestCountKey := packetReassemblyIndexKeyFactory(p, common.StringOrNil(packetReassemblyFragmentIngestCountKeySuffix))
	if ingestCountKey == nil {
		return false, nil, errors.New("Failed to ingest packet fragment; packet reassembly key factory returned nil fragment ingest count key")
	}
	i, err := redisutil.Increment(*ingestCountKey)
	if err != nil {
		return false, nil, err
	}

	ingestCount := uint(*i)
	return verified, &ingestCount, nil
}

// Verify the fragment checksum
func (p *packetFragment) Verify() (bool, error) {
	if p.Checksum == nil {
		return false, errors.New("failed to validate fragment; nil checksum")
	}

	if len(*p.Checksum) != 16 { // require 128-bit checksum
		return false, fmt.Errorf("failed to validate fragment; 128-bit checksum required for verification (%d-bit checksum found: %s)", len(*p.Checksum)*8, *p.Checksum)
	}

	if p.Payload == nil {
		return false, errors.New("failed to validate fragment; nil payload")
	}

	var checksumBytes [16]byte
	copy(checksumBytes[:], *p.Checksum)
	return (checksumBytes == md5.Sum([]byte(*p.Payload))), nil
}

// packetReassembly represents a NATS packet reassembly message payload
type packetReassembly struct {
	Cardinality uint    `json:"cardinality"` // i.e., # of total fragments comprising the packet
	Checksum    *string `json:"checksum"`    // i.e., md5 checksum of the entire n of n payload
	Next        *string `json:"next"`        // i.e., the "next" hop, represented currently as a subject where the handling of the reconstituted packet will terminate; the message broadcast to the "next" hop will receive a pointer to the reassembled packet
	Nonce       *int64  `json:"nonce"`       // i.e., nonce to prevent replay attacks (accidental or otherwise)
	Size        uint    `json:"size"`        // i.e., size of the reassembled packet
	Payload     *[]byte `json:"-"`           // i.e., memory address where the reconstituted packet can be optionally read

	// TODO: forward secrecy & key negotiation (i.e., diffie-hellman)
}

// fragmentIngestProgress calculates and returns the fragment ingest progress (expressed as a percentage), the
// total number of fragments ingested and any error if the attempt to retrieve or calculate the progress fails
func (p *packetReassembly) fragmentIngestProgress() (*float64, *uint, error) {
	ingestCountKey := packetReassemblyIndexKeyFactory(p, common.StringOrNil(packetReassemblyFragmentIngestCountKeySuffix))
	if ingestCountKey == nil {
		return nil, nil, fmt.Errorf("Failed to reassemble packet with checksum %s; reassembly key factory returned nil key", *p.Checksum)
	}

	var i uint

	rawval, err := redisutil.Get(*ingestCountKey)
	if err == nil && rawval != nil {
		intval, converr := strconv.Atoi(*rawval)
		if converr == nil {
			i = uint(intval)
		} else {
			err = converr
		}
	} else if rawval == nil {
		err = fmt.Errorf("Retrieved nil value from redis for packet reassembly ingest count key: %s", *ingestCountKey)
	}

	if err != nil {
		return nil, nil, err
	}

	progress := float64(i) / float64(p.Cardinality)
	return &progress, &i, nil
}

// Broadcast marshals and transmits the packet reassembly header payload
func (p *packetReassembly) Broadcast() error {
	payload, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return natsutil.NatsStreamingPublish(natsPacketReassembleSubject, payload)
}

// Cache the packet reassembly as a header (i.e., without its payload)
func (p *packetReassembly) Cache() error {
	headerKey := packetReassemblyIndexKeyFactory(p, common.StringOrNil(packetReassemblyHeaderKeySuffix))
	if headerKey == nil {
		return errors.New("Failed to cache reassembly header; packet reassembly key factory returned nil header key")
	}

	payload, _ := json.Marshal(p)
	return redisutil.Set(*headerKey, string(payload), nil)
}

// Reassemble defrags the packet and verifies the checksum of the reconstituted packet
func (p *packetReassembly) Reassemble() (bool, error) {
	if p.Checksum == nil {
		return false, errors.New("no packet reassembly attempted; nil checksum")
	}

	if len(*p.Checksum) != 16 { // require 128-bit checksum
		return false, fmt.Errorf("no packet reassembly attempted; 128-bit checksum required for verification (%d-bit checksum found: %s)", len(*p.Checksum)*8, *p.Checksum)
	}

	if p.Cardinality == 0 {
		return false, errors.New("no packet reassembly attempted; cardinality must be greater than zero")
	}

	if p.Size == 0 {
		return false, errors.New("no packet reassembly attempted; size must be greater than zero")
	}

	p.Payload = nil // side-effect of this method sets payload pointer

	fragSize := p.Size / p.Cardinality
	progress, i, err := p.fragmentIngestProgress()
	if err != nil {
		return false, fmt.Errorf("Failed to reassemble %d-byte packet consisting of %d %d-byte fragment(s); failed atomically reading or parsing fragment ingest progress; %s", p.Size, p.Cardinality, fragSize, err.Error())
	}
	if *progress < 1 {
		return false, fmt.Errorf("Failed to reassemble %d-byte packet consisting of %d %d-byte fragment(s); %d (%f%%) of required fragments ingested", p.Size, p.Cardinality, fragSize, *i, *progress)
	}

	common.Log.Debugf("All %d fragments required to reassemble %d-byte packet with checksum %s have been ingested; attempting reassembly and verification...", p.Cardinality, p.Size, *p.Checksum)
	payload := make([]byte, p.Size)

	common.Log.Debugf("incomplete Reassemble() implementation... TODO: reassemble %d fragments", p.Cardinality)

	p.Payload = &payload
	verified, err := p.Verify()
	if err != nil || !verified {
		msg := fmt.Sprintf("reassembly or checksum verification failed")
		if err != nil {
			msg = fmt.Sprintf("%s; %s", msg, err.Error())
		}
		return false, fmt.Errorf("Failed to reassemble packet with checksum %s; %s", *p.Checksum, msg)
	}

	return verified, nil
}

// Verify the reassembled packet checksum
func (p *packetReassembly) Verify() (bool, error) {
	if p.Checksum == nil {
		return false, errors.New("failed to validate fragment; nil checksum")
	}

	if len(*p.Checksum) != 16 { // require 128-bit checksum
		return false, fmt.Errorf("failed to validate reassembled packet; 128-bit checksum required for verification (%d-bit checksum found: %s)", len(*p.Checksum)*8, *p.Checksum)
	}

	if p.Payload == nil {
		return false, errors.New("failed to validate reassembled packet; nil payload")
	}

	var checksumBytes [16]byte
	copy(checksumBytes[:], *p.Checksum)
	return (checksumBytes == md5.Sum(*p.Payload)), nil
}
