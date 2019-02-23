#!/usr/bin/env python3
"""
Copyright (c) 2016-present, Facebook, Inc.
All rights reserved.

This source code is licensed under the BSD-style license found in the
LICENSE file in the root directory of this source tree. An additional grant
of patent rights can be found in the PATENTS file in the same directory.

Pre-run script for services to generate a nghttpx config from a jinja template
and the config/mconfig for the service.
"""

import logging
import socket

import os
from create_oai_certs import generate_mme_certs
from generate_service_config import generate_template_config
from orc8r.protos import common_pb2

from magma.common.misc_utils import get_ip_from_if, get_ip_from_if_cidr
from magma.configuration.mconfig_managers import load_service_mconfig
from magma.configuration.service_configs import get_service_config_value

CONFIG_OVERRIDE_DIR = '/var/opt/magma/tmp'


def _get_iface_ip(service, iface_config):
    """
    Get the interface IP given its name.
    """
    iface_name = get_service_config_value(service, iface_config, "")
    return get_ip_from_if_cidr(iface_name)


def _get_dns_ip(iface_config):
    """
    Get dnsd interface IP without netmask.
    If caching is enabled, use the ip of interface that dnsd listens over.
    Otherwise, just use dns server in yml.
    """
    if load_service_mconfig('mme').enable_dns_caching:
        iface_name = get_service_config_value('dnsd', iface_config, '')
        return get_ip_from_if(iface_name)
    return get_service_config_value('spgw', 'ipv4_dns', '')


def _get_oai_log_level():
    """
    Convert the logLevel in mconfig into the level which OAI code
    uses. We use OAI's 'TRACE' as the debugging log level and 'CRITICAL'
    as the fatal log level.
    """
    mconfig = load_service_mconfig('mme')
    oai_log_level = 'INFO'
    if mconfig.log_level == common_pb2.DEBUG:
        oai_log_level = 'TRACE'
    elif mconfig.log_level == common_pb2.INFO:
        oai_log_level = 'INFO'
    elif mconfig.log_level == common_pb2.WARNING:
        oai_log_level = 'WARNING'
    elif mconfig.log_level == common_pb2.ERROR:
        oai_log_level = 'ERROR'
    elif mconfig.log_level == common_pb2.FATAL:
        oai_log_level = 'CRITICAL'
    return oai_log_level


def _get_relay_enabled():
    if load_service_mconfig('mme').relay_enabled:
        return "yes"
    return "no"


def _get_non_eps_service_control():
    if (load_service_mconfig('mme')
            .non_eps_service_control):
        if (load_service_mconfig('mme')
                .non_eps_service_control == 0):
            return "OFF"
        elif (load_service_mconfig('mme')
                .non_eps_service_control == 1):
            return "CSFB_SMS"
        elif (load_service_mconfig('mme')
                .non_eps_service_control == 2):
            return "SMS"
    return "OFF"


def _get_lac():
    if load_service_mconfig('mme').lac:
        return load_service_mconfig('mme').lac
    return 0


def _get_csfb_mcc():
    if load_service_mconfig('mme').csfb_mcc:
        return load_service_mconfig('mme').csfb_mcc
    return ""


def _get_csfb_mnc():
    if load_service_mconfig('mme').csfb_mnc:
        return load_service_mconfig('mme').csfb_mnc
    return ""


def _get_context():
    """
    Create the context which has the interface IP and the OAI log level to use.
    """
    context = {}
    context['s11_ip'] = _get_iface_ip('spgw', 's11_iface_name')
    context['s1ap_ip'] = _get_iface_ip('mme', 's1ap_iface_name')
    context['s1u_ip'] = _get_iface_ip('spgw', 's1u_iface_name')
    context['oai_log_level'] = _get_oai_log_level()
    context['ipv4_dns'] = _get_dns_ip('dns_iface_name')
    realm = get_service_config_value('mme', 'realm', "")
    context['identity'] = socket.gethostname() + '.' + realm
    context['relay_enabled'] = _get_relay_enabled()
    context['non_eps_service_control'] = _get_non_eps_service_control()
    context['csfb_mcc'] = _get_csfb_mcc()
    context['csfb_mnc'] = _get_csfb_mnc()
    context['lac'] = _get_lac()
    # set ovs params
    for key in ('ovs_bridge_name', 'ovs_gtp_port_number',
                'ovs_uplink_port_number', 'ovs_uplink_mac'):
        context[key] = get_service_config_value('spgw', key, '')
    return context


def main():
    logging.basicConfig(
        level=logging.INFO,
        format='[%(asctime)s %(levelname)s %(name)s] %(message)s')
    context = _get_context()
    generate_template_config('spgw', 'spgw',
                             CONFIG_OVERRIDE_DIR, context.copy())
    generate_template_config('mme', 'mme', CONFIG_OVERRIDE_DIR, context.copy())
    generate_template_config('mme', 'mme_fd',
                             CONFIG_OVERRIDE_DIR, context.copy())
    cert_dir = get_service_config_value('mme', 'cert_dir', "")
    generate_mme_certs(os.path.join(cert_dir, 'freeDiameter'))


if __name__ == "__main__":
    main()