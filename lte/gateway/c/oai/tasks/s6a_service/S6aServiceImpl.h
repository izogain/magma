/*
 * Licensed to the OpenAirInterface (OAI) Software Alliance under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The OpenAirInterface Software Alliance licenses this file to You under
 * the Apache License, Version 2.0  (the "License"); you may not use this file
 * except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *-------------------------------------------------------------------------------
 * For more information about the OpenAirInterface (OAI) Software Alliance:
 *      contact@openairinterface.org
 */
#pragma once

#include <grpc++/grpc++.h>
#include "lte/protos/s6a_service.grpc.pb.h"

using grpc::ServerContext;
using grpc::Status;
using magma::lte::DeleteSubscriberRequest;
using magma::lte::DeleteSubscriberResponse;
using magma::lte::S6aService;

namespace magma {
using namespace lte;

class S6aServiceImpl final : public S6aService::Service {
 public:
  S6aServiceImpl();

  /*
       * Deletes the subscribers in the DeleteSubscriberRequest.
       *
       * @param context: the grpc Server context
       * @param request: deleteSubscriberRequest, contains a list of IMSI of
                        subscriber to delete.
       * @param response (out): the DeleteSubscriberResponse that contains
                                err message.
       * @return grpc Status instance
       */
  Status DeleteSubscriber(
    ServerContext *context,
    const DeleteSubscriberRequest *request,
    DeleteSubscriberResponse *response) override;
};

} // namespace magma
