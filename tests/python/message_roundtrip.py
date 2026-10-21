#!/usr/bin/env python3

from google.protobuf.descriptor import FieldDescriptor

from xgc.registry.v1 import message_registry
from xgc.robot.v1 import message_pb2 as robot_message_pb2
from xgc.robot.v1 import server_pb2
from xgc.semantic.aerial.v1 import control_pb2
from xgc.semantic.common.v1 import acceleration_pb2
from xgc.semantic.ground.v1 import control_pb2 as ground_control_pb2
from xgc.semantic.ground.v1 import locomotion_pb2
from xgc.semantic.ground.v1 import reset_pb2
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
        3105: (
            "xgc.semantic.ground.v1.ResetStatus",
            1,
            4388678013613103954,
            reset_pb2.ResetStatus,
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


def assert_reset_status_roundtrip():
    assert reset_pb2.RESET_PHASE_UNSPECIFIED == 0
    assert reset_pb2.RESET_PHASE_IDLE == 1
    assert reset_pb2.RESET_PHASE_DONE == 6
    assert reset_pb2.RESET_OUTCOME_ARRIVED == 1
    assert reset_pb2.RESET_OUTCOME_TIMEOUT == 2
    assert reset_pb2.RESET_OUTCOME_STOPPED == 3
    assert reset_pb2.RESET_OUTCOME_REJECTED_NO_TARGET == 10
    assert reset_pb2.RESET_OUTCOME_REJECTED_NO_COORDINATOR == 11
    assert reset_pb2.RESET_OUTCOME_REJECTED_BY_COORDINATOR == 12
    assert reset_pb2.RESET_OUTCOME_INFEASIBLE == 20
    assert reset_pb2.RESET_OUTCOME_ABORTED_DEVIATION == 30
    assert reset_pb2.RESET_OUTCOME_ABORTED_STALLED == 31
    assert reset_pb2.RESET_OUTCOME_ABORTED_COORDINATOR_LOST == 32
    assert reset_pb2.RESET_OUTCOME_ABORTED_UNHEALTHY == 33
    assert reset_pb2.RESET_REASON_NONE == 0
    assert reset_pb2.RESET_REASON_SCENE_CHANGED == 26
    assert reset_pb2.ResetResult.DESCRIPTOR.containing_type is None
    assert reset_pb2.ResetStatus.DESCRIPTOR.nested_types == []
    status = reset_pb2.ResetStatus(
        generation=7,
        phase=reset_pb2.RESET_PHASE_DONE,
        result=reset_pb2.ResetResult(
            outcome=reset_pb2.RESET_OUTCOME_ARRIVED,
            reason=reset_pb2.RESET_REASON_GOAL_OCCUPIED,
            final_planar_error_m=0.04,
            final_yaw_error_rad=0.02,
            duration_s=31.0,
            detail="goal occupied by ugv3",
            source_stamp_ns=123,
        ),
    )
    decoded = reset_pb2.ResetStatus.FromString(status.SerializeToString())
    assert decoded == status
    assert message_registry.METADATA[3105]["fingerprint"] == 4388678013613103954
    registered = message_registry.new_message(3105)
    assert isinstance(registered, reset_pb2.ResetStatus)
    registered.ParseFromString(status.SerializeToString())
    assert registered.generation == 7
    assert registered.result.source_stamp_ns == 123


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
    assert_reset_status_roundtrip()
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
