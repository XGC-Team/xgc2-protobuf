#include <cassert>
#include <iostream>
#include <memory>
#include <string>

#include "xgc/registry/v1/message_registry.hpp"
#include "xgc/robot/v1/message.pb.h"
#include "xgc/robot/v1/server.pb.h"
#include "xgc/semantic/aerial/v1/control.pb.h"
#include "xgc/semantic/common/v1/acceleration.pb.h"
#include "xgc/semantic/ground/v1/control.pb.h"
#include "xgc/semantic/ground/v1/locomotion.pb.h"
#include "xgc/v1/message.pb.h"

int main() {
  const auto* empty_metadata = xgc::registry::v1::findMessage(1);
  assert(empty_metadata != nullptr);
  assert(empty_metadata->version == 1u);
  assert(empty_metadata->fingerprint == 11009224659857530918ULL);
  assert(std::string(empty_metadata->full_name) == "xgc.v1.Empty");
  assert(dynamic_cast<xgc::v1::Empty*>(xgc::registry::v1::newMessage(1).get()) != nullptr);
  assert(dynamic_cast<xgc::semantic::ground::v1::LocomotionStatus*>(
             xgc::registry::v1::newMessage(3103).get()) != nullptr);
  assert(dynamic_cast<xgc::semantic::ground::v1::JointStateSet*>(
             xgc::registry::v1::newMessage(3104).get()) != nullptr);
  assert(dynamic_cast<xgc::semantic::common::v1::AccelerationEstimate*>(
             xgc::registry::v1::newMessage(2008).get()) != nullptr);
  const auto* locomotion_metadata = xgc::registry::v1::findMessage(3103);
  assert(locomotion_metadata != nullptr);
  assert(locomotion_metadata->fingerprint == 17502343282573601552ULL);
  const auto* joints_metadata = xgc::registry::v1::findMessage(3104);
  assert(joints_metadata != nullptr);
  assert(joints_metadata->fingerprint == 12421060107119594740ULL);

  const auto* spec_metadata = xgc::registry::v1::findMessage(4001);
  assert(spec_metadata != nullptr);
  assert(spec_metadata->version == 3u);
  assert(spec_metadata->fingerprint == 2292867660820935957ULL);
  assert(std::string(spec_metadata->full_name) == "xgc.robot.v1.RobotAdapterSpec");
  assert(dynamic_cast<xgc::robot::v1::RobotAdapterSpec*>(
             xgc::registry::v1::newMessage(4001).get()) != nullptr);

  const auto* routed_metadata = xgc::registry::v1::findMessage(4002);
  assert(routed_metadata != nullptr);
  assert(routed_metadata->version == 1u);
  assert(routed_metadata->fingerprint == 17732826818852005547ULL);
  assert(std::string(routed_metadata->full_name) == "xgc.robot.v1.RobotMessage");
  assert(dynamic_cast<xgc::robot::v1::RobotMessage*>(
             xgc::registry::v1::newMessage(4002).get()) != nullptr);

  xgc::semantic::aerial::v1::ModeRequest request;
  request.set_mode("OFFBOARD");

  std::string payload;
  assert(request.SerializeToString(&payload));

  const auto* metadata = xgc::registry::v1::findMessage(3202);
  assert(metadata != nullptr);

  xgc::v1::Message envelope;
  envelope.set_sequence(1);
  envelope.mutable_payload()->mutable_schema()->set_message_id(metadata->id);
  envelope.mutable_payload()->mutable_schema()->set_type_name(metadata->full_name);
  envelope.mutable_payload()->mutable_schema()->set_schema_version(metadata->version);
  envelope.mutable_payload()->mutable_schema()->set_schema_fingerprint(metadata->fingerprint);
  envelope.mutable_payload()->set_encoding(xgc::v1::PAYLOAD_ENCODING_PROTOBUF);
  envelope.mutable_payload()->set_value(payload);

  xgc::robot::v1::RobotMessage routed;
  routed.set_robot_id("uav1");
  routed.set_channel_id("operation.mode");
  *routed.mutable_message() = envelope;

  std::unique_ptr<google::protobuf::Message> decoded =
      xgc::registry::v1::newMessage(envelope.payload().schema().message_id());
  assert(decoded != nullptr);
  assert(decoded->ParseFromString(envelope.payload().value()));

  const auto* mode = dynamic_cast<const xgc::semantic::aerial::v1::ModeRequest*>(decoded.get());
  assert(mode != nullptr);
  assert(mode->mode() == "OFFBOARD");

  assert(dynamic_cast<xgc::semantic::aerial::v1::ArmRequest*>(
             xgc::registry::v1::newMessage(3201).get()) != nullptr);
  assert(dynamic_cast<xgc::semantic::aerial::v1::AutopilotRebootRequest*>(
             xgc::registry::v1::newMessage(3203).get()) != nullptr);
  assert(dynamic_cast<xgc::semantic::aerial::v1::ForceDisarmRequest*>(
             xgc::registry::v1::newMessage(3206).get()) != nullptr);
  const auto* ground_intent_metadata = xgc::registry::v1::findMessage(3204);
  assert(ground_intent_metadata != nullptr);
  assert(ground_intent_metadata->fingerprint == 13602409479439522314ULL);
  assert(dynamic_cast<xgc::semantic::ground::v1::MotionIntentRequest*>(
             xgc::registry::v1::newMessage(3204).get()) != nullptr);
  assert(xgc::registry::v1::newMessage(5001) == nullptr);

  // Describe is the readiness contract of Run-owned robot type servers: the
  // ROS1 type servers implement it in C++ and Core calls it with wait_ready_ms.
  xgc::robot::v1::DescribeRequest describe_request;
  describe_request.set_wait_ready_ms(30000);
  std::string describe_request_bytes;
  assert(describe_request.SerializeToString(&describe_request_bytes));
  xgc::robot::v1::DescribeRequest parsed_request;
  assert(parsed_request.ParseFromString(describe_request_bytes));
  assert(parsed_request.wait_ready_ms() == 30000u);

  xgc::robot::v1::DescribeResponse describe_response;
  describe_response.mutable_service_ref()->set_instance_id("instance");
  describe_response.set_ready(true);
  describe_response.set_facts_json(
      "{\"ros_master_uri\":\"http://127.0.0.1:11311\",\"ros_master_run_id\":\"run\"}");
  std::string describe_response_bytes;
  assert(describe_response.SerializeToString(&describe_response_bytes));
  xgc::robot::v1::DescribeResponse parsed_response;
  assert(parsed_response.ParseFromString(describe_response_bytes));
  assert(parsed_response.ready());
  assert(parsed_response.service_ref().instance_id() == "instance");
  assert(parsed_response.facts_json() == describe_response.facts_json());

  // Peers from before readiness stay compatible: an empty request means no
  // wait, and a reply without the new fields is not ready.
  xgc::robot::v1::DescribeRequest legacy_request;
  assert(legacy_request.ParseFromString(""));
  assert(legacy_request.wait_ready_ms() == 0u);
  xgc::robot::v1::DescribeResponse legacy_response;
  assert(legacy_response.ParseFromString(std::string("\x12\x01p", 3)));  // provider "p"
  assert(legacy_response.provider_definition_id() == "p");
  assert(!legacy_response.ready() && legacy_response.facts_json().empty());

  std::cout << "cpp roundtrip ok: " << routed.robot_id() << " " << routed.channel_id()
            << " mode=" << mode->mode() << std::endl;
  return 0;
}
