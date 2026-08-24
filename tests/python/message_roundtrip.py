#!/usr/bin/env python3

from xgc.registry.v1 import message_registry
from xgc.robot.v1 import message_pb2 as robot_message_pb2
from xgc.semantic.aerial.v1 import control_pb2
from xgc.semantic.common.v1 import acceleration_pb2
from xgc.semantic.ground.v1 import control_pb2 as ground_control_pb2
from xgc.semantic.ground.v1 import locomotion_pb2
from xgc.v1 import message_pb2


def assert_domain_boundary_registry():
    expected = {
        1: ("xgc.v1.Empty", 1, 11009224659857530918, message_pb2.Empty),
        4001: (
            "xgc.robot.v1.RobotAdapterSpec",
            3,
            2292867660820935957,
            robot_message_pb2.RobotAdapterSpec,
        ),
        4002: (
            "xgc.robot.v1.RobotMessage",
            1,
            17732826818852005547,
            robot_message_pb2.RobotMessage,
        ),
        3103: (
            "xgc.semantic.ground.v1.LocomotionStatus",
            1,
            17502343282573601552,
            locomotion_pb2.LocomotionStatus,
        ),
        3104: (
            "xgc.semantic.ground.v1.JointStateSet",
            1,
            12421060107119594740,
            locomotion_pb2.JointStateSet,
        ),
        2008: (
            "xgc.semantic.common.v1.AccelerationEstimate",
            1,
            4994560962731678057,
            acceleration_pb2.AccelerationEstimate,
        ),
    }
    for message_id, (full_name, version, fingerprint, message_type) in expected.items():
        metadata = message_registry.METADATA[message_id]
        assert metadata["id"] == message_id
        assert metadata["full_name"] == full_name
        assert metadata["version"] == version
        assert metadata["fingerprint"] == fingerprint
        assert isinstance(message_registry.new_message(message_id), message_type)


def main():
    assert_domain_boundary_registry()
    payload = control_pb2.ModeRequest(mode="OFFBOARD").SerializeToString()
    metadata = message_registry.METADATA[3202]
    envelope = message_pb2.Message(
        sequence=1,
        payload=message_pb2.Payload(
            schema=message_pb2.SchemaReference(
                message_id=metadata["id"],
                type_name=metadata["full_name"],
                schema_version=metadata["version"],
                schema_fingerprint=metadata["fingerprint"],
            ),
            encoding=message_pb2.PAYLOAD_ENCODING_PROTOBUF,
            value=payload,
        ),
    )
    routed = robot_message_pb2.RobotMessage(
        robot_id="uav1", channel_id="operation.mode", message=envelope
    )
    decoded = message_registry.new_message(envelope.payload.schema.message_id)
    decoded.ParseFromString(envelope.payload.value)
    assert decoded.mode == "OFFBOARD"
    assert isinstance(message_registry.new_message(3201), control_pb2.ArmRequest)
    assert (
        message_registry.new_message(3103).DESCRIPTOR.full_name
        == "xgc.semantic.ground.v1.LocomotionStatus"
    )
    assert (
        message_registry.new_message(3104).DESCRIPTOR.full_name
        == "xgc.semantic.ground.v1.JointStateSet"
    )
    assert isinstance(
        message_registry.new_message(3203), control_pb2.AutopilotRebootRequest
    )
    ground_intent = message_registry.new_message(3204)
    assert isinstance(ground_intent, ground_control_pb2.MotionIntentRequest)
    assert message_registry.METADATA[3204]["fingerprint"] == 13602409479439522314
    assert message_registry.new_message(5001) is None
    assert message_registry.new_message(999999) is None
    print(
        "python roundtrip ok: {} {} mode={}".format(
            routed.robot_id, routed.channel_id, decoded.mode
        )
    )


if __name__ == "__main__":
    main()
