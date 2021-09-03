package consumer

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/kthomas/go-redisutil"
	"github.com/provideplatform/nchain/common"
	"github.com/vmihailenco/msgpack/v5"
)

const packetReassemblyHeaderKeySuffix = "header"
const packetReassemblyFragmentIngestCountKeySuffix = "fragments.ingest-count"
const packetReassemblyFragmentPersistenceKeySuffix = "fragments.persistence"

const packetReassemblyFragmentationChunkSize = uint(4500)

// Fragmentable interface
type Fragmentable interface {
	// Broadcast marshals and transmits the fragment metadata and payload
	Broadcast(network INetwork, wg *sync.WaitGroup) error

	// Ingest verifies the checksum of the fragment, writes the the underlying bytes to persistent
	// or ephemeral storage, atomically increments the internal ingest counter associated with the
	// packet reassembly operation and returns the boolean checksum verification result, total number
	// of fragments ingested (i.e., after ingesting this fragment-- or nil, if ingestion failed) and
	// any error if the attempt to ingest the fragment fails
	Ingest(db IDatabase) (bool, *uint, error)

	// Verify the fragment checksum
	Verify() (bool, error)
}

// BroadcastFragments splits the given packet into chunks. Each chunk is broadcast as a fragment for remote packet
// reassembly. If `usePadding` is true then the last packet sent will be padded if required to make sure that all
// packets sent have an equal size.
func BroadcastFragments(network INetwork, packet []byte, usePadding bool, wg *sync.WaitGroup) error {
	packetSize := uint(len(packet))
	chunkSize := packetReassemblyFragmentationChunkSize

	numChunks := uint(1)
	if packetSize > chunkSize {
		numChunks = uint(math.Ceil(float64(packetSize) / float64(chunkSize)))
	}

	PacketReassembly := packetReassemblyFactory(packet, numChunks)
	fragments := make([]*packetFragment, numChunks)

	index := uint(0)
	bytesRemaining := uint(len(packet))

	common.Log.Debugf("Chunking %d-byte packet with checksum %s into %d %d-byte chunks", packetSize, hex.EncodeToString(*PacketReassembly.Checksum), numChunks, chunkSize)
	for index <= numChunks-1 {
		start := index * chunkSize
		end := start + chunkSize
		padChunk := false

		if bytesRemaining < chunkSize {
			end = start + bytesRemaining

			if usePadding {
				padChunk = true
			}
		}

		var fragmentPayload []byte
		if padChunk {
			fragmentPayload = make([]byte, chunkSize)
			copy(fragmentPayload[:], packet[start:end])
		} else {
			fragmentPayload = packet[start:end]
		}

		checksum := md5.Sum(fragmentPayload)
		checksumSlice := checksum[:]

		fragments[index] = &packetFragment{
			Index:               index,
			Cardinality:         numChunks,
			Checksum:            &checksumSlice,
			Nonce:               PacketReassembly.Nonce,
			Payload:             &fragmentPayload,
			ReassembledChecksum: PacketReassembly.Checksum,
		}

		bytesRemaining = bytesRemaining - chunkSize
		index++
	}

	common.Log.Debugf("Prepared %d packet fragments for broadcast", len(fragments))
	if wg != nil {
		wg.Add(1)
	}
	go PacketReassembly.Broadcast(network, wg)
	for _, fragment := range fragments {
		if wg != nil {
			wg.Add(1)
		}
		go fragment.Broadcast(network, wg)
	}

	return nil
}

// PacketReassemblyFactory constructs a new packet reassembly instance which can be used
// as a header within the first fragment of a distributed packet reassembly operation
func packetReassemblyFactory(packet []byte, cardinality uint) *packetReassembly {
	checksum := md5.Sum(packet)
	checksumSlice := checksum[:]
	nonce := time.Now().UnixNano()

	return &packetReassembly{
		Cardinality: cardinality,
		Checksum:    &checksumSlice,
		Nonce:       &nonce,
		Size:        uint(len(packet)),
		Payload:     &packet,
	}
}

// packetReassemblyIndexKeyFactory returns a unique identifier for in-memory cache & mutexes,
// fragment persistent storage facilities, etc. for in-flight packet reassembly operations
// for the given Fragmentable
func packetReassemblyIndexKeyFactory(fragmentable Fragmentable, suffix *string) *string {
	var checksum *[]byte
	var nonce *int64

	switch packet := fragmentable.(type) {
	case *packetFragment:
		checksum = packet.ReassembledChecksum
		nonce = packet.Nonce
	case *packetReassembly:
		checksum = packet.Checksum
		nonce = packet.Nonce
	default:
		common.Log.Warning("Reflection not supported for given fragmentable")
	}

	if checksum == nil {
		common.Log.Warningf("Failed to resolve checksum of fragmentable during attempt to construct index key")
		return nil
	}

	digest := sha256.New()
	digest.Write([]byte(fmt.Sprintf("%s.%d.%s", packetReassembleSubject, *nonce, *checksum)))
	key := hex.EncodeToString(digest.Sum(nil))

	if suffix != nil {
		key = fmt.Sprintf("%s.%s", key, *suffix)
	}

	return &key
}

func fragmentIndexKeyFactory(nonce *int64, index uint, reassembledChecksum *[]byte, suffix *string) *string {
	digest := sha256.New()
	key := fmt.Sprintf("%s.%d.%s.%d", packetFragmentIngestSubject, *nonce, hex.EncodeToString(*reassembledChecksum), index)
	digest.Write([]byte(key))
	key = hex.EncodeToString(digest.Sum(nil))

	if suffix != nil {
		key = fmt.Sprintf("%s.%s", key, *suffix)
	}

	return &key
}

// packetFragment represents a packet fragment ingest message payload; TODO: support marshaling from wire/protocol in addition to JSON
type packetFragment struct {
	Index               uint    `msgpack:"index"`                // the index of the fragment
	Cardinality         uint    `msgpack:"cardinality"`          // # of total fragments comprising the packet
	Checksum            *[]byte `msgpack:"checksum"`             // md5 checksum of the fragment payload
	Nonce               *int64  `msgpack:"nonce"`                // nonce associated with the packet reassembly operation
	Payload             *[]byte `msgpack:"payload,inline"`       // the raw fragment payload
	ReassembledChecksum *[]byte `msgpack:"reassembled_checksum"` // md5 checksum of the entire n of n payload

	// TODO: forward secrecy considerations
}

// Broadcast marshals and transmits the fragment metadata and payload
func (p *packetFragment) Broadcast(network INetwork, wg *sync.WaitGroup) error {
	if wg != nil {
		defer wg.Done()
	}

	payload, err := msgpack.Marshal(p)
	if err != nil {
		return err
	}

	common.Log.Tracef("attempting to broadcast %d-byte fragment for Nonce %d - index #%d", len(payload), p.Nonce, p.Index)
	_, err = network.Send(packetFragmentIngestSubject, payload)
	return err
}

// FetchReassemblyHeader fetches the previously-cached packet reassembly header, warms the fragment-local
// `Reassembly` reference and returns the loaded `PacketReassembly` -- or returns nil and an error if the
// header failed to load for any reason.
func (p *packetFragment) FetchReassemblyHeader(db IDatabase) (*packetReassembly, error) {
	headerKey := packetReassemblyIndexKeyFactory(p, common.StringOrNil(packetReassemblyHeaderKeySuffix))
	if headerKey == nil {
		return nil, errors.New("failed to fetch cached reassembly header; packet reassembly key factory returned nil header key")
	}

	rawval, err := db.Get(*headerKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cached reassembly header for ; %s", err.Error())
	} else if rawval == nil {
		return nil, fmt.Errorf("failed to parse valid reassembly header; received nil value from cache")
	}

	var reassembly *packetReassembly
	err = msgpack.Unmarshal([]byte(*rawval), &reassembly)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached reassembly header; %s", err.Error())
	}

	return reassembly, nil
}

// Ingest verifies the checksum of the fragment, writes the the underlying bytes to persistent
// or ephemeral storage, atomically increments the internal ingest counter associated with the
// packet reassembly operation and returns the boolean checksum verification result, total number
// of fragments ingested (i.e., after ingesting this fragment-- or nil, if ingestion failed) and
// any error if the attempt to ingest the fragment fails
func (p *packetFragment) Ingest(db IDatabase) (bool, *uint, error) {
	if p.Checksum == nil {
		return false, nil, errors.New("no fragment ingestion attempted; nil checksum")
	}

	payload, _ := msgpack.Marshal(p)

	if len(*p.Checksum) != 16 { // require 128-bit checksum
		return false, nil, fmt.Errorf("no fragment ingestion attempted; 128-bit checksum required for verification (%d-bit checksum found: %s)", len(*p.Checksum)*8, *p.Checksum)
	}

	verified, err := p.Verify()
	if err != nil || !verified {
		msg := "fragment ingest or checksum verification failed"
		if err != nil {
			msg = fmt.Sprintf("%s; %s", msg, err.Error())
		}
		return false, nil, errors.New(msg)
	}

	persistKey := fragmentIndexKeyFactory(p.Nonce, p.Index, p.ReassembledChecksum, common.StringOrNil(packetReassemblyFragmentPersistenceKeySuffix))
	if persistKey == nil {
		return false, nil, errors.New("failed to cache packet data; fragment index key factory returned nil persist key")
	}
	err = db.Set(*persistKey, string(payload), nil)
	if err != nil {
		return false, nil, err
	}

	ingestCountKey := packetReassemblyIndexKeyFactory(p, common.StringOrNil(packetReassemblyFragmentIngestCountKeySuffix))
	if ingestCountKey == nil {
		return false, nil, errors.New("failed to ingest packet fragment; packet reassembly key factory returned nil fragment ingest count key")
	}
	i, err := db.Increment(*ingestCountKey)
	if err != nil || i == nil {
		// TODO: if this fails but the cache of the payload worked?
		return false, nil, err
	}

	ingestCount := uint(*i)

	return verified, &ingestCount, nil
}

// Verify the fragment checksum
func (p *packetFragment) Verify() (bool, error) {
	// Check for null data
	if p.Checksum == nil {
		return false, errors.New("failed to validate fragment; nil checksum")
	}
	if p.ReassembledChecksum == nil {
		return false, errors.New("failed to validate fragment; nil reassembled checksum")
	}
	if p.Nonce == nil {
		return false, errors.New("failed to validate fragment; nil nonce")
	}
	if p.Checksum == nil {
		return false, errors.New("failed to validate fragment; nil checksum")
	}
	if p.Payload == nil {
		return false, errors.New("failed to validate fragment; nil payload")
	}

	// Check for bad data
	if p.Cardinality == 0 {
		return false, errors.New("failed to validate fragment; cardinality is zero")
	}
	if p.Index >= p.Cardinality {
		return false, fmt.Errorf("failed to validate fragment; invalid index (%d) greater than cardinality (%d)", p.Index, p.Cardinality)
	}

	// Verify checksum
	if len(*p.Checksum) != 16 { // require 128-bit checksum
		return false, fmt.Errorf("failed to validate fragment; 128-bit checksum required for verification (%d-bit checksum found: %s)", len(*p.Checksum)*8, *p.Checksum)
	}
	var checksumBytes [16]byte
	copy(checksumBytes[:], *p.Checksum)
	return (checksumBytes == md5.Sum([]byte(*p.Payload))), nil
}

// packetReassembly represents a NATS packet reassembly message payload
type packetReassembly struct {
	Cardinality uint    `msgpack:"cardinality"` // i.e., # of total fragments comprising the packet
	Checksum    *[]byte `msgpack:"checksum"`    // i.e., md5 checksum of the entire n of n payload
	Nonce       *int64  `msgpack:"nonce"`       // i.e., nonce to prevent replay attacks (accidental or otherwise)
	Size        uint    `msgpack:"size"`        // i.e., size of the reassembled packet
	Payload     *[]byte `msgpack:"-"`           // i.e., memory address where the reconstituted packet can be optionally read

	// TODO: forward secrecy & key negotiation (i.e., diffie-hellman)
}

// fragmentIngestProgress calculates and returns the fragment ingest progress (expressed as a percentage), the
// total number of fragments ingested and any error if the attempt to retrieve or calculate the progress fails
func (p *packetReassembly) fragmentIngestProgress() (*float64, *uint, error) {
	ingestCountKey := packetReassemblyIndexKeyFactory(p, common.StringOrNil(packetReassemblyFragmentIngestCountKeySuffix))
	if ingestCountKey == nil {
		return nil, nil, fmt.Errorf("Failed to reassemble packet with checksum %s; reassembly key factory returned nil key", *p.Checksum)
	}

	var count uint

	rawval, err := redisutil.Get(*ingestCountKey)
	if err == nil && rawval != nil {
		intval, converr := strconv.Atoi(*rawval)
		if converr == nil {
			count = uint(intval)
		} else {
			err = converr
		}
	} else if rawval == nil {
		err = fmt.Errorf("Retrieved nil value from redis for packet reassembly ingest count key: %s", *ingestCountKey)
	}

	if err != nil {
		return nil, nil, err
	}

	progress := float64(count) / float64(p.Cardinality+1)
	return &progress, &count, nil
}

// Broadcast marshals and transmits the packet reassembly header payload
func (p *packetReassembly) Broadcast(network INetwork, wg *sync.WaitGroup) error {
	if wg != nil {
		defer wg.Done()
	}

	payload, err := msgpack.Marshal(p)
	if err != nil {
		return err
	}
	_, err = network.Send(packetReassembleSubject, payload)
	return err
}

// Ingest and cache the packet reassembly as a header (i.e., without its payload)
func (p *packetReassembly) Ingest(db IDatabase) (bool, *uint, error) {
	headerKey := packetReassemblyIndexKeyFactory(p, common.StringOrNil(packetReassemblyHeaderKeySuffix))
	if headerKey == nil {
		return false, nil, errors.New("failed to ingest reassembly header; packet reassembly key factory returned nil header key")
	}
	payload, _ := msgpack.Marshal(p)
	err := db.Set(*headerKey, string(payload), nil)
	if err != nil {
		return false, nil, err
	}

	ingestCountKey := packetReassemblyIndexKeyFactory(p, common.StringOrNil(packetReassemblyFragmentIngestCountKeySuffix))
	if ingestCountKey == nil {
		return false, nil, errors.New("failed to ingest packet fragment; packet reassembly key factory returned nil fragment ingest count key")
	}
	i, err := db.Increment(*ingestCountKey)
	if err != nil || i == nil {
		// TODO: if this fails but the cache of the payload worked?
		return false, nil, err
	}

	ingestCount := uint(*i)
	return true, &ingestCount, nil
}

// Reassemble defrags the packet and verifies the checksum of the reconstituted packet
func (p *packetReassembly) Reassemble(db IDatabase) (bool, error) {
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

	progress, _, err := p.fragmentIngestProgress()
	if err != nil {
		return false, fmt.Errorf("failed to reassemble %d-byte packet consisting of %d fragment(s); failed atomically reading or parsing fragment ingest progress; %s", p.Size, p.Cardinality, err.Error())
	}
	if *progress < 1 {
		return false, fmt.Errorf("failed to reassemble %d-byte packet consisting of %d fragment(s); only %.2f%% of packets ingested", p.Size, p.Cardinality, *progress)
	}

	common.Log.Debugf("All %d fragments required to reassemble %d-byte packet with checksum %s have been ingested; attempting reassembly and verification...", p.Cardinality, p.Size, hex.EncodeToString(*p.Checksum))
	payload := make([]byte, p.Size)

	lastIndex := uint(0)
	for i := uint(0); i < p.Cardinality; i++ {
		// Get fragment from storage
		persistKey := fragmentIndexKeyFactory(p.Nonce, i, p.Checksum, common.StringOrNil(packetReassemblyFragmentPersistenceKeySuffix))
		data, err := db.Get(*persistKey)
		if err != nil {
			return false, fmt.Errorf("failed to get packet index %d with key '%s' from storage", i, *persistKey)
		}

		// Decode fragment
		fragment := &packetFragment{}
		err = msgpack.Unmarshal([]byte(*data), &fragment)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal packet index %d with key '%s' from storage", i, *persistKey)
		}

		payloadSize := uint(len(*fragment.Payload))
		remainingBytes := p.Size - lastIndex
		if remainingBytes < payloadSize {
			// Don't copy padding
			payloadSize = remainingBytes
		}

		copy(payload[lastIndex:lastIndex+payloadSize], *fragment.Payload)
		lastIndex += payloadSize
	}

	p.Payload = &payload
	verified, err := p.Verify()
	if err != nil || !verified {
		msg := "reassembly or checksum verification failed"
		if err != nil {
			msg = fmt.Sprintf("%s; %s", msg, err.Error())
		}
		return false, fmt.Errorf("failed to reassemble packet with checksum %s; %s", *p.Checksum, msg)
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
