package reassembly

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"github.com/kthomas/go-redisutil"
)

func setEnvVarIfNotExists(name string, value string) {
	existing := os.Getenv(name)
	if existing == "" {
		os.Setenv(name, value)
	}
}

func setupRedis() {
	setEnvVarIfNotExists("REDIS_HOSTS", "127.0.0.1:6379")
	setEnvVarIfNotExists("REDIS_DB_INDEX", "1")
	setEnvVarIfNotExists("REDIS_PASSWORD", "test")

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
	fragment := &PacketFragment{}
	err := json.Unmarshal(msg, &fragment)

	if err != nil {
		t.Errorf("Unable to decode message (call #%d)", call)
		return
	}

	// Check that fragment had a payload & decode it into raw bytes
	var decodedPayload []byte
	if fragment.Payload == nil {
		t.Errorf("Fragment contained no Payload, (call #%d)", call)
	} else {
		var derr error
		decodedPayload, derr = base64.StdEncoding.DecodeString(*fragment.Payload)
		if derr != nil {
			t.Errorf("Unable to b64 decode Payload (call #%d)", call)
		}
		actualSize := uint(len(decodedPayload))
		if actualSize != fragment.PayloadSize {
			t.Errorf("Incorrect fragment packet size. Payload was %d bytes, but Size set to %d", actualSize, fragment.PayloadSize)
		}
	}

	// Check that the checksum is present & correct
	if fragment.Checksum == nil {
		t.Errorf("Fragment contained no checksum, (call #%d)", call)
	} else {
		_, derr := base64.StdEncoding.DecodeString(*fragment.Checksum)
		if derr != nil {
			t.Errorf("Unable to b64 decode Checksum (call #%d)", call)
		}

		// Manually check the checksum
		// Hash the decoded payload & encode in b64 to compare with Checksum
		hashBytes := md5.Sum(decodedPayload)
		expectedHash := base64.StdEncoding.EncodeToString(hashBytes[:])

		if *fragment.Checksum != expectedHash {
			t.Errorf("Fragment hash was not as expected. (call #%d)", call)
		}
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

func checkReassemblyIsValid(t *testing.T, msg []byte) *PacketReassembly {
	// Decode data into a fragment
	reassembly := &PacketReassembly{}
	err := json.Unmarshal(msg, &reassembly)

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

	setupRedis()

	// Stub out the publish function so that BroadcastFragments will use our stub above to "send" data.
	var callsToPublish uint64 = 0
	var reassembly *PacketReassembly = nil
	SetBroadcastPublishFunction(func(subject string, msg []byte) error {
		callsToPublish++

		if len(msg) > 7000 {
			t.Errorf("Large fragment found. Expected size: <= 7000, Actual: %d", len(msg))
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

	if reassembly == nil {
		t.Error("BroadcastFragments() error; did not get reassembly packet")
		return
	}

	// Reassemble the packets
	ass_ret, ass_err := reassembly.Reassemble()
	if !ass_ret {
		t.Error("reassembly.Reassemble() returned false")
	}
	if ass_err != nil {
		t.Errorf("reassembly.Reassemble() error; %s", ass_err.Error())
	}

	// While we might not care about how specifically the packets are broken down - for the large test file
	// we expect that it is broken down into a large number of packets, so the streamingPublish method should
	// be called more than 10000 times for this file.
	if 5000 > callsToPublish {
		t.Errorf("BroadcastFragments() not called correct amount of times. Expected: over 10000, Actual: %d.", callsToPublish)
	}
}

func loadPacket(t *testing.T, path string) *PacketFragment {
	data := string(getTestFile(t, path))

	var fragment *PacketFragment
	err := json.Unmarshal([]byte(data), &fragment)
	if err != nil {
		t.Error("test error; unable to unserialise 'test/valid-packet.json' PacketFragment")
	}

	return fragment
}

func expectFragmentDecodes(t *testing.T, fragment *PacketFragment) {
	err := fragment.Decode()
	if err != nil {
		t.Errorf("Decode() error; Decode returned error, expected no error. err: '%s'.", err)
	}
}

func expectFragmentIsValid(t *testing.T, fragment *PacketFragment) {
	valid, err := fragment.Verify()
	if !valid {
		t.Errorf("Verify() error; Verify returned false, expected true, err: '%s'.", err)
	}
	if err != nil {
		t.Errorf("Verify() error; Verify returned error, expected no error. err: '%s'.", err)
	}
}

func expectFragmentIsNotValid(t *testing.T, fragment *PacketFragment, expectedError error) {
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
		expectFragmentDecodes(t, fragment)
		expectFragmentIsValid(t, fragment)
	}

	// Wrong Checksum
	{
		fragment := loadPacket(t, "test/invalid-packet-wrong-checksum.json")
		expectFragmentDecodes(t, fragment)
		expectFragmentIsNotValid(t, fragment, nil)
	}

	// Can be extended with further invalid packet types, such as Cardinality = 0, or Index > Cardinality, etc.

}
