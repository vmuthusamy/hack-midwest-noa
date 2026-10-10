"""Transport contract tests. These use a stub generator, not a model quality eval."""
import asyncio
import unittest
from concurrent import futures
from unittest.mock import patch
import grpc
from agent_service.server import EvidenceService, pb, rpc

class AgentTransportTests(unittest.TestCase):
    def setUp(self):
        async def generator(evidence):
            self.input=evidence
            return {"changed_facts":[],"source_references":[],"evidence_gaps":["No prior approval"],"explanation":"Test transport response","draft_operational_messages":[]}
        self.service=EvidenceService(generator)
        self.server=grpc.server(futures.ThreadPoolExecutor(max_workers=2))
        rpc.add_EvidenceBriefingServiceServicer_to_server(self.service,self.server)
        port=self.server.add_insecure_port('127.0.0.1:0');self.server.start()
        self.channel=grpc.insecure_channel('127.0.0.1:'+str(port));self.client=rpc.EvidenceBriefingServiceStub(self.channel)
    def tearDown(self):
        self.channel.close();self.server.stop(0)
    def request(self):
        return pb.ExplainRequest(request_id='request-1',mode='LIVE',facility=pb.Facility(id='SC-DEMO-01',fictional=True),evidence_timestamp='2026-10-10T21:07:32Z',alerts=[pb.Alert(id='source-1',instruction='Official untrusted data',message_type='Alert')])
    def test_typed_snapshot_identity_and_mapping(self):
        with patch('agent_service.server.configured',return_value=True): result=self.client.Explain(self.request(),timeout=2)
        self.assertEqual(result.request_id,'request-1');self.assertEqual(result.facility_id,'SC-DEMO-01')
        self.assertEqual(self.input['alerts'][0]['instruction'],'Official untrusted data')
        self.assertEqual(self.input['alerts'][0]['messageType'],'Alert')
        self.assertEqual(list(result.briefing.draft_operational_messages),[])
    def test_unconfigured_provider_is_explicit(self):
        with patch('agent_service.server.configured',return_value=False),self.assertRaises(grpc.RpcError) as error:self.client.Explain(self.request(),timeout=2)
        self.assertEqual(error.exception.code(),grpc.StatusCode.FAILED_PRECONDITION)
    def test_deadline_cancels_generator_and_releases_worker(self):
        async def slow(evidence):await asyncio.sleep(30)
        self.service.generator=slow
        with patch('agent_service.server.configured',return_value=True):
            with self.assertRaises(grpc.RpcError) as error:self.client.Explain(self.request(),timeout=.05)
            self.assertEqual(error.exception.code(),grpc.StatusCode.DEADLINE_EXCEEDED)
            import time
            deadline=time.monotonic()+2
            while self.service.lock.locked() and time.monotonic()<deadline:time.sleep(.01)
            self.assertFalse(self.service.lock.locked())
