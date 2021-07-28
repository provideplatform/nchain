package consumer

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"math/rand"
	"os"
	"testing"

	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/consumer"
)

func setupRedis() {
	redisutil.RequireRedis()
}

func generateRandomBytes(length int) []byte {
	data := make([]byte, length)
	for i := range data {
		data[i] = byte(rand.Intn(255))
	}
	return data
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
		t.Errorf("Fragment Ingest returned false. (call #%d)", call)
	}
	if verr != nil {
		t.Errorf("Fragment Ingest encountered error: '%s' (call #%d)", verr, call)
	}
}

func checkReassemblyIsValid(t *testing.T, msg []byte) *packetReassembly {
	// Decode data into a reassembly packet
	reassembly := &packetReassembly{}
	err := msgpack.Unmarshal(msg, &reassembly)

	if err != nil {
		t.Errorf("Unable to reassembly packet")
		return nil
	}

	// Ingest the fragment
	valid, _, verr := reassembly.Ingest()
	if !valid {
		t.Error("Reassembly Ingest returned false")
	}
	if verr != nil {
		t.Errorf("Reassembly Ingest encountered error: '%s'", verr)
	}

	return reassembly
}

func TestBroadcastFragments(t *testing.T) {
	payload := generateRandomBytes(1024 * 128)
	setupRedis()

	// Stub out the publish function so that BroadcastFragments will use our stub above to "send" data.
	var callsToPublish uint64 = 0
	var reassembly *packetReassembly = nil
	var totalMsgSize uint = 0
	setBroadcastPublishFunction(func(subject string, msg []byte) (*string, error) {
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

		return nil, nil
	})

	// Run the actual fragment broadcast
	err := BroadcastFragments(payload, true)
	if err != nil {
		t.Errorf("BroadcastFragments() error; %s", err.Error())
	}

	// Reassemble the packets
	if reassembly == nil {
		t.Error("BroadcastFragments() error; did not get reassembly packet")
		return
	}
	ret, err := reassembly.Reassemble()
	if !ret {
		t.Error("reassembly.Reassemble() returned false")
	}
	if err != nil {
		t.Errorf("reassembly.Reassemble() error; %s", err.Error())
	}

	// Note: this may change as packet size or test data size is tweaked.
	if callsToPublish != 31 {
		t.Errorf("BroadcastFragments() not called correct amount of times. Expected: 31, Actual: %d.", callsToPublish)
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
	valid, err := fragment.Verify()
	if valid {
		t.Errorf("Verify() error; Verify returned true, expected false.")
	}
	if err != expectedError {
		t.Errorf("Verify() error; Verify error not as expected. Actual: '%s'. Expected: '%s'.", err, expectedError)
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
