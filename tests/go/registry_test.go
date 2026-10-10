package registryv1

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	adapterv1 "xgc2/protocols/xgc/adapter/v1"
	robotv1 "xgc2/protocols/xgc/robot/v1"
	aerialv1 "xgc2/protocols/xgc/semantic/aerial/v1"
	commonv1 "xgc2/protocols/xgc/semantic/common/v1"
	groundv1 "xgc2/protocols/xgc/semantic/ground/v1"
	xgcv1 "xgc2/protocols/xgc/v1"
)

func TestAerialOperationRoundTripThroughMessage(t *testing.T) {
	payload, err := proto.Marshal(&aerialv1.ModeRequest{Mode: "OFFBOARD"})
	if err != nil {
		t.Fatal(err)
	}
	metadata, ok := Lookup(3202)
	if !ok {
		t.Fatal("mode request is not registered")
	}
	envelope := &xgcv1.Message{
		Sequence: 1,
		Payload: &xgcv1.Payload{
			Schema: &xgcv1.SchemaReference{
				MessageId: metadata.ID, TypeName: metadata.FullName,
				SchemaVersion: metadata.Version, SchemaFingerprint: metadata.Fingerprint,
			},
			Encoding: xgcv1.PayloadEncoding_PAYLOAD_ENCODING_PROTOBUF,
			Value:    payload,
		},
	}
	routed := &robotv1.RobotMessage{RobotId: "uav1", ChannelId: "operation.mode", Message: envelope}
	if routed.GetMessage().GetSequence() != 1 {
		t.Fatal("robot routing wrapper lost the generic message")
	}
	decoded, ok := New(envelope.GetPayload().GetSchema().GetMessageId())
	if !ok {
		t.Fatal("registered message could not be constructed")
	}
	if err := proto.Unmarshal(envelope.GetPayload().GetValue(), decoded); err != nil {
		t.Fatal(err)
	}
	request, ok := decoded.(*aerialv1.ModeRequest)
	if !ok || request.GetMode() != "OFFBOARD" {
		t.Fatalf("unexpected decoded payload: %#v", decoded)
	}
}

func TestRobotAdapterSpecIsTypedDomainConfiguration(t *testing.T) {
	spec := &robotv1.RobotAdapterSpec{
		RobotSelectionDigest: "robot-selection-digest",
		Robots: []*robotv1.RobotResource{{
			RobotId: "robot1", ProfileId: "fixture.robot-profile.v1", ProfileDigest: "profile-digest",
			Parameters: map[string]string{"namespace": "/uav1"},
			Channels:   []*robotv1.ChannelGrant{{ChannelId: "state.pose", Enabled: true}},
		}},
	}
	encoded, err := proto.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	payload := &xgcv1.Payload{
		Schema:   &xgcv1.SchemaReference{TypeName: "xgc.robot.v1.RobotAdapterSpec", SchemaVersion: 3},
		Encoding: xgcv1.PayloadEncoding_PAYLOAD_ENCODING_PROTOBUF,
		Value:    encoded,
	}
	instanceSpec := &adapterv1.AdapterInstanceSpec{
		InstanceId: "robot-adapter", ProcessGeneration: 1, Revision: 1,
		SpecDigest: "spec-digest", Configuration: payload,
	}
	var decoded robotv1.RobotAdapterSpec
	if err := proto.Unmarshal(instanceSpec.GetConfiguration().GetValue(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.GetRobotSelectionDigest() != "robot-selection-digest" || len(decoded.GetRobots()) != 1 ||
		decoded.GetRobots()[0].GetChannels()[0].GetChannelId() != "state.pose" {
		t.Fatalf("unexpected robot Adapter config: %v", &decoded)
	}
}

func TestChannelGrantOwnsOnlyChannelSelection(t *testing.T) {
	descriptor := (&robotv1.ChannelGrant{}).ProtoReflect().Descriptor()
	if descriptor.Fields().ByName("parameters") != nil {
		t.Fatal("ChannelGrant must not carry runtime parameters")
	}
	if descriptor.Fields().Len() != 2 {
		t.Fatalf("ChannelGrant has unexpected fields: %d", descriptor.Fields().Len())
	}
	reserved := descriptor.ReservedRanges()
	if reserved.Len() != 1 || reserved.Get(0)[0] != 3 || reserved.Get(0)[1] != 4 {
		t.Fatalf("ChannelGrant field 3 must stay reserved: %v", reserved)
	}
}

func TestDomainBoundaryMessagesAreRegistered(t *testing.T) {
	tests := []struct {
		id          uint32
		version     uint32
		fingerprint uint64
		fullName    string
		message     proto.Message
	}{
		{
			id: 1, version: 1, fingerprint: 11009224659857530918,
			fullName: "xgc.v1.Empty", message: &xgcv1.Empty{},
		},
		{
			id: 4001, version: 3, fingerprint: 2292867660820935957,
			fullName: "xgc.robot.v1.RobotAdapterSpec", message: &robotv1.RobotAdapterSpec{},
		},
		{
			id: 4002, version: 1, fingerprint: 17732826818852005547,
			fullName: "xgc.robot.v1.RobotMessage", message: &robotv1.RobotMessage{},
		},
		{
			id: 3103, version: 1, fingerprint: 17502343282573601552,
			fullName: "xgc.semantic.ground.v1.LocomotionStatus", message: &groundv1.LocomotionStatus{},
		},
		{
			id: 3104, version: 1, fingerprint: 12421060107119594740,
			fullName: "xgc.semantic.ground.v1.JointStateSet", message: &groundv1.JointStateSet{},
		},
		{
			id: 2008, version: 1, fingerprint: 4994560962731678057,
			fullName: "xgc.semantic.common.v1.AccelerationEstimate", message: &commonv1.AccelerationEstimate{},
		},
	}
	for _, test := range tests {
		metadata, ok := Lookup(test.id)
		if !ok {
			t.Fatalf("message ID %d is not registered", test.id)
		}
		if metadata.ID != test.id || metadata.Version != test.version ||
			metadata.Fingerprint != test.fingerprint || metadata.FullName != test.fullName {
			t.Fatalf("message ID %d has unexpected metadata: %#v", test.id, metadata)
		}
		created, ok := New(test.id)
		if !ok {
			t.Fatalf("message ID %d cannot be constructed", test.id)
		}
		if created.ProtoReflect().Descriptor().FullName() != test.message.ProtoReflect().Descriptor().FullName() {
			t.Fatalf("message ID %d resolved to %s", test.id, created.ProtoReflect().Descriptor().FullName())
		}
	}
}

func TestOperationMessagesAreRegistered(t *testing.T) {
	tests := []struct {
		id      uint32
		message proto.Message
	}{
		{id: 3201, message: &aerialv1.ArmRequest{Armed: true}},
		{id: 3202, message: &aerialv1.ModeRequest{}},
		{id: 3203, message: &aerialv1.AutopilotRebootRequest{}},
		{id: 3204, message: &groundv1.MotionIntentRequest{Gear: 2, Longitudinal: 1, Yaw: -1}},
		{id: 3206, message: &aerialv1.ForceDisarmRequest{}},
	}
	for _, test := range tests {
		created, ok := New(test.id)
		if !ok {
			t.Fatalf("message ID %d is not registered", test.id)
		}
		if created.ProtoReflect().Descriptor().FullName() != test.message.ProtoReflect().Descriptor().FullName() {
			t.Fatalf("message ID %d resolved to %s", test.id, created.ProtoReflect().Descriptor().FullName())
		}
	}
	if _, ok := New(5001); ok {
		t.Fatal("raw MAVLink request ID must remain unavailable")
	}
	if _, ok := New(999999); ok {
		t.Fatal("unknown message ID should stay opaque")
	}
}

func TestPx4DiagnosticMessagesAreRegistered(t *testing.T) {
	tests := []struct {
		id      uint32
		message proto.Message
	}{
		{id: 3002, message: &aerialv1.LocalTrajectorySetpoint{}},
		{id: 3003, message: &aerialv1.AttitudeSetpoint{}},
		{id: 3004, message: &aerialv1.FcuLinkStatus{}},
		{id: 3005, message: &aerialv1.OffboardInputStatus{}},
	}
	for _, test := range tests {
		created, ok := New(test.id)
		if !ok {
			t.Fatalf("message ID %d is not registered", test.id)
		}
		if created.ProtoReflect().Descriptor().FullName() != test.message.ProtoReflect().Descriptor().FullName() {
			t.Fatalf("message ID %d resolved to %s", test.id, created.ProtoReflect().Descriptor().FullName())
		}
	}
}

// Describe is the readiness contract of Run-owned robot type servers. Field
// numbers are wire identity: the first seven response fields predate readiness
// and must never move, and the instance fence stays in service_ref.
func TestRobotServerDescribeCarriesReadinessContract(t *testing.T) {
	type field struct {
		name   protoreflect.Name
		number protoreflect.FieldNumber
		kind   protoreflect.Kind
	}
	tests := []struct {
		message proto.Message
		fields  []field
	}{
		{&robotv1.DescribeRequest{}, []field{
			{"wait_ready_ms", 1, protoreflect.Uint32Kind},
		}},
		{&robotv1.DescribeResponse{}, []field{
			{"service_ref", 1, protoreflect.MessageKind},
			{"provider_definition_id", 2, protoreflect.StringKind},
			{"maximum_robots", 3, protoreflect.Uint32Kind},
			{"maximum_members", 4, protoreflect.Uint32Kind},
			{"maximum_batch_members", 5, protoreflect.Uint32Kind},
			{"maximum_commands_per_robot", 6, protoreflect.Uint32Kind},
			{"runtime_policy", 7, protoreflect.MessageKind},
			{"ready", 8, protoreflect.BoolKind},
			{"facts_json", 9, protoreflect.StringKind},
		}},
	}
	for _, test := range tests {
		descriptor := test.message.ProtoReflect().Descriptor()
		if descriptor.Fields().Len() != len(test.fields) {
			t.Fatalf("%s has %d fields, want %d", descriptor.FullName(), descriptor.Fields().Len(), len(test.fields))
		}
		for _, want := range test.fields {
			got := descriptor.Fields().ByName(want.name)
			if got == nil || got.Number() != want.number || got.Kind() != want.kind {
				t.Fatalf("%s.%s must be field %d of kind %s", descriptor.FullName(), want.name, want.number, want.kind)
			}
		}
	}
	reference := (&robotv1.DescribeResponse{}).ProtoReflect().Descriptor().Fields().ByName("service_ref").Message()
	if reference.Fields().ByName("instance_id") == nil {
		t.Fatal("DescribeResponse.service_ref must carry the instance fence")
	}

	reply := &robotv1.DescribeResponse{
		ServiceRef: &adapterv1.RuntimeServiceReference{InstanceId: "instance"},
		Ready:      true,
		FactsJson:  `{"ros_master_uri":"http://127.0.0.1:11311","ros_master_run_id":"run"}`,
	}
	encoded, err := proto.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	var decoded robotv1.DescribeResponse
	if err := proto.Unmarshal(encoded, &decoded); err != nil || !proto.Equal(reply, &decoded) {
		t.Fatalf("readiness reply did not round-trip: %v", err)
	}
	encoded, err = proto.Marshal(&robotv1.DescribeRequest{WaitReadyMs: 30000})
	if err != nil {
		t.Fatal(err)
	}
	var request robotv1.DescribeRequest
	if err := proto.Unmarshal(encoded, &request); err != nil || request.GetWaitReadyMs() != 30000 {
		t.Fatalf("readiness request did not round-trip: %v", err)
	}

	// Peers from before readiness stay compatible: an empty request means no
	// wait, and a reply without the new fields is not ready.
	var legacyRequest robotv1.DescribeRequest
	if err := proto.Unmarshal(nil, &legacyRequest); err != nil || legacyRequest.GetWaitReadyMs() != 0 {
		t.Fatalf("empty request must mean no wait: %v", err)
	}
	var legacyReply robotv1.DescribeResponse
	providerOnly := []byte{0x12, 0x01, 'p'} // provider_definition_id = "p"
	if err := proto.Unmarshal(providerOnly, &legacyReply); err != nil ||
		legacyReply.GetProviderDefinitionId() != "p" || legacyReply.GetReady() || legacyReply.GetFactsJson() != "" {
		t.Fatalf("reply without readiness fields must decode as not ready: %v", err)
	}
}
