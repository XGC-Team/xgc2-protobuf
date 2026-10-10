#!/usr/bin/env python3

from google.protobuf.descriptor import FieldDescriptor

from xgc.registry.v1 import message_registry
from xgc.robot.v1 import message_pb2 as robot_message_pb2
from xgc.robot.v1 import server_pb2
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


def assert_robot_server_readiness_contract():
    # Field numbers are wire identity: the first seven response fields predate
    # readiness and must never move, and the instance fence stays in service_ref.
    def fields(message_type):
        return {
            field.name: (field.number, field.type)
            for field in message_type.DESCRIPTOR.fields
        }

    assert fields(server_pb2.DescribeRequest) == {
        "wait_ready_ms": (1, FieldDescriptor.TYPE_UINT32),
    }
    assert fields(server_pb2.DescribeResponse) == {
        "service_ref": (1, FieldDescriptor.TYPE_MESSAGE),
        "provider_definition_id": (2, FieldDescriptor.TYPE_STRING),
        "maximum_robots": (3, FieldDescriptor.TYPE_UINT32),
        "maximum_members": (4, FieldDescriptor.TYPE_UINT32),
        "maximum_batch_members": (5, FieldDescriptor.TYPE_UINT32),
        "maximum_commands_per_robot": (6, FieldDescriptor.TYPE_UINT32),
        "runtime_policy": (7, FieldDescriptor.TYPE_MESSAGE),
        "ready": (8, FieldDescriptor.TYPE_BOOL),
        "facts_json": (9, FieldDescriptor.TYPE_STRING),
    }
    assert "instance_id" in server_pb2.DescribeResponse.DESCRIPTOR.fields_by_name[
        "service_ref"
    ].message_type.fields_by_name

    reply = server_pb2.DescribeResponse(
        ready=True,
        facts_json='{"ros_master_uri":"http://127.0.0.1:11311","ros_master_run_id":"run"}',
    )
    reply.service_ref.instance_id = "instance"
    assert server_pb2.DescribeResponse.FromString(reply.SerializeToString()) == reply
    request = server_pb2.DescribeRequest(wait_ready_ms=30000)
    assert server_pb2.DescribeRequest.FromString(request.SerializeToString()) == request

    # Peers from before readiness stay compatible: an empty request means no
    # wait, and a reply without the new fields is not ready.
    assert server_pb2.DescribeRequest.FromString(b"").wait_ready_ms == 0
    legacy = server_pb2.DescribeResponse.FromString(b"\x12\x01p")  # provider "p"
    assert legacy.provider_definition_id == "p"
    assert not legacy.ready and legacy.facts_json == ""


def main():
    assert_domain_boundary_registry()
    assert_robot_server_readiness_contract()
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
    assert isinstance(
        message_registry.new_message(3206), control_pb2.ForceDisarmRequest
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
