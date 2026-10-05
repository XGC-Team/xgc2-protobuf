package registryv1

import (
	"testing"

	"google.golang.org/protobuf/proto"
	adapterv1 "xgc2/protocols/xgc/adapter/v1"
	robotv1 "xgc2/protocols/xgc/robot/v1"
	aerialv1 "xgc2/protocols/xgc/semantic/aerial/v1"
	commonv1 "xgc2/protocols/xgc/semantic/common/v1"
	groundv1 "xgc2/protocols/xgc/semantic/ground/v1"
	xgcv1 "xgc2/protocols/xgc/v1"
	worldv1 "xgc2/protocols/xgc/world/v1"
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
		Robot: &robotv1.RobotResource{
			RobotId: "robot1", ProfileId: "fixture.robot-profile.v1", ProfileDigest: "profile-digest",
			Parameters: map[string]string{"namespace": "/uav1"},
			Channels:   []*robotv1.ChannelGrant{{ChannelId: "state.pose", Enabled: true}},
		},
	}
	encoded, err := proto.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	metadata, ok := Lookup(4001)
	if !ok {
		t.Fatal("robot configuration is not registered")
	}
	payload := &xgcv1.Payload{
		Schema: &xgcv1.SchemaReference{
			MessageId: metadata.ID, TypeName: metadata.FullName,
			SchemaVersion: metadata.Version, SchemaFingerprint: metadata.Fingerprint,
		},
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
	if decoded.GetRobotSelectionDigest() != "robot-selection-digest" || decoded.GetRobot() == nil ||
		decoded.GetRobot().GetChannels()[0].GetChannelId() != "state.pose" {
		t.Fatalf("unexpected robot Adapter config: %v", &decoded)
	}
	descriptor := decoded.ProtoReflect().Descriptor()
	if descriptor.Fields().ByNumber(2) != nil || !descriptor.ReservedRanges().Has(2) ||
		!descriptor.ReservedNames().Has("robots") || descriptor.Fields().ByName("robot").Number() != 3 {
		t.Fatal("single robot must use tag 3 and retire the robots field/tag 2")
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
			id: 4001, version: 4, fingerprint: 5254957371271658330,
			fullName: "xgc.robot.v1.RobotAdapterSpec", message: &robotv1.RobotAdapterSpec{},
		},
		{
			id: 4002, version: 1, fingerprint: 2768611942346717735,
			fullName: "xgc.robot.v1.RobotMessage", message: &robotv1.RobotMessage{},
		},
		{
			id: 4003, version: 1, fingerprint: 8026494345279606082,
			fullName: "xgc.world.v1.WorldRuntimeSpec", message: &worldv1.WorldRuntimeSpec{},
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
