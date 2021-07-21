package consumer

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"os"
	"testing"

	"github.com/kthomas/go-redisutil"
	"github.com/provideplatform/nchain/common"
	"github.com/vmihailenco/msgpack/v5"
)

func setupRedis() {
	redisutil.RequireRedis()
}

func getTestFile(t *testing.T, path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Unable to open test file: '%s'.", path)
	}
	return data
}

func checkFragmentIsValid(t *testing.T, msg []byte, call uint64) {
	// Decode data into a fragment
	fragment := &packetFragment{}
	err := msgpack.Unmarshal(msg, &fragment)

	if err != nil {
		t.Errorf("Unable to decode message (call #%d)", call)
		return
	}

	// Check that fragment has a payload
	if fragment.Payload == nil {
		t.Errorf("Fragment contained no Payload, (call #%d)", call)
		return
	}

	actualSize := uint(len(*fragment.Payload))
	if actualSize != fragment.PayloadSize {
		t.Errorf("Incorrect fragment packet size. Payload was %d bytes, but Size set to %d", actualSize, fragment.PayloadSize)
	}

	// Check that the checksum is present & correct
	if fragment.Checksum == nil {
		t.Errorf("Fragment contained no checksum, (call #%d)", call)
		return
	}

	// Manually check the checksum
	// Hash the Payload & compare with Checksum
	hashBytes := md5.Sum(*fragment.Payload)

	if !bytes.Equal(*fragment.Checksum, hashBytes[:]) {
		t.Errorf("Fragment hash was not as expected. (call #%d)", call)
	}

	// Ensure Index is within Cardinality range
	if fragment.Index >= fragment.Cardinality {
		t.Errorf("Fragment index was out of bounds, (call #%d). Index: %d, Cardinality: %d",
			call, fragment.Index, fragment.Cardinality)
	}

	// Ingest the fragment
	valid, _, verr := fragment.Ingest()
	if !valid {
		t.Errorf("Fragment verify returned false. (call #%d)", call)
	}
	if verr != nil {
		t.Errorf("Fragment verify encountered error: '%s' (call #%d)", verr, call)
	}
}

func checkReassemblyIsValid(t *testing.T, msg []byte) *packetReassembly {
	// Decode data into a fragment
	reassembly := &packetReassembly{}
	err := msgpack.Unmarshal(msg, &reassembly)

	if err != nil {
		t.Errorf("Unable to reassembly packet")
		return nil
	}

	reassembly.Cache()
	return reassembly
}

func TestBroadcastFragments(t *testing.T) {
	// Load a large test file to use as a binary blob to broadcast
	payload := getTestFile(t, "test/test1.bin")
	payloadLength := uint(len(payload))
	setupRedis()

	// Stub out the publish function so that BroadcastFragments will use our stub above to "send" data.
	var callsToPublish uint64 = 0
	var reassembly *packetReassembly = nil
	var totalMsgSize uint = 0
	SetBroadcastPublishFunction(func(subject string, msg []byte) error {
		callsToPublish++

		length := uint(len(msg))
		totalMsgSize += length

		if length > 7000 {
			t.Errorf("Large fragment found. Expected size: <= 7000, Actual: %d. subject: '%s'", len(msg), subject)
		}

		// Subject is always natsPacketFragmentIngestSubject?
		if subject == natsPacketFragmentIngestSubject {
			checkFragmentIsValid(t, msg, callsToPublish)
		} else if subject == natsPacketReassembleSubject {
			if reassembly != nil {
				t.Error("Recieved more than one reassembly packet")
			}
			reassembly = checkReassemblyIsValid(t, msg)
		} else {
			t.Errorf("Unknown fragment subject. Subject: '%s'", subject)
		}

		return nil
	})

	// Run the actual fragment broadcast
	err := BroadcastFragments(payload)
	if err != nil {
		t.Errorf("BroadcastFragments() error; %s", err.Error())
	}

	// Check size
	common.Log.Debugf("Payload Length: %d, Total Bytes Sent: %d, Overhead: %.1f%%", payloadLength, totalMsgSize, (1-(float32(payloadLength)/float32(totalMsgSize)))*100)

	// Reassemble the packets
	if reassembly == nil {
		t.Error("BroadcastFragments() error; did not get reassembly packet")
		return
	}
	assRet, assErr := reassembly.Reassemble()
	if !assRet {
		t.Error("reassembly.Reassemble() returned false")
	}
	if assErr != nil {
		t.Errorf("reassembly.Reassemble() error; %s", assErr.Error())
	}

	// While we might not care about how specifically the packets are broken down - for the large test file
	// we expect that it is broken down into a large number of packets, so the streamingPublish method should
	// be called more than 10000 times for this file.
	if 5000 > callsToPublish {
		t.Errorf("BroadcastFragments() not called correct amount of times. Expected: over 10000, Actual: %d.", callsToPublish)
	}
}

func loadPacket(t *testing.T, path string) *packetFragment {
	data := string(getTestFile(t, path))

	var fragment *packetFragment
	err := json.Unmarshal([]byte(data), &fragment)
	if err != nil {
		t.Errorf("test error; unable to unserialise '%s' PacketFragment", path)
	}

	return fragment
}

func expectFragmentIsValid(t *testing.T, fragment *packetFragment) {
	valid, err := fragment.Verify()
	if !valid {
		t.Errorf("Verify() error; Verify returned false, expected true, err: '%s'.", err)
	}
	if err != nil {
		t.Errorf("Verify() error; Verify returned error, expected no error. err: '%s'.", err)
	}
}

func expectFragmentIsNotValid(t *testing.T, fragment *packetFragment, expectedError error) {
	valid, verr := fragment.Verify()
	if valid {
		t.Errorf("Verify() error; Verify returned true, expected false.")
	}
	if verr != expectedError {
		t.Errorf("Verify() error; Verify error not as expected. Actual: '%s'. Expected: '%s'.", verr, expectedError)
	}
}

func TestIngest(t *testing.T) {

	setupRedis()

	// Valid packet = success
	{
		fragment := loadPacket(t, "test/valid-packet.json")
		ingested, count, err := fragment.Ingest()
		if !ingested {
			t.Errorf("Ingest() error; Ingest returned false, expected true, err: '%s'.", err)
		}
		if count == nil {
			t.Error("Ingest() error; Ingest count not as expected non-nil, got nil")
		}
		if err != nil {
			t.Errorf("Ingest() error; Ingest err not as expected, expected nil, actual: '%s'.", err)
		}
	}

	// Ingest invalid packet = fail
	{
		fragment := loadPacket(t, "test/invalid-packet-wrong-checksum.json")
		ingested, count, err := fragment.Ingest()
		if ingested {
			t.Error("Ingest() error; Ingest returned true, expected false")
		}
		if count != nil {
			t.Errorf("Ingest() error; Ingest expcted nil count, got %d", count)
		}
		if err == nil {
			t.Errorf("Ingest() error; Ingest err not as expected, expected non-nil, actual: nil")
		}
	}

}

func TestVerify(t *testing.T) {

	// Valid packet
	{
		fragment := loadPacket(t, "test/valid-packet.json")
		expectFragmentIsValid(t, fragment)
	}

	// Wrong Checksum
	{
		fragment := loadPacket(t, "test/invalid-packet-wrong-checksum.json")
		expectFragmentIsNotValid(t, fragment, nil)
	}

	// Can be extended with further invalid packet types, such as Cardinality = 0, or Index > Cardinality, etc.

}
