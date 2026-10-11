import json
import unittest
from datetime import datetime, timezone
from pathlib import Path
from unittest.mock import patch
import sys
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from evidence import normalize, geography, FACILITY, fetch_evidence
from briefing import Briefing, ChangedFact, validate_output, build_input, AGENT_INSTRUCTIONS

ROOT=Path(__file__).resolve().parents[1]
FEATURE=json.loads((ROOT/'fixtures/noaa-warning-recorded.json').read_text())['feature']
ACTIVE=datetime(2026,10,10,17,40,tzinfo=timezone.utc)

class EvidenceTests(unittest.TestCase):
    def test_recorded_polygon_contains_fictional_facility(self):
        self.assertEqual(geography(FEATURE['geometry']), 'applicable')
    def test_outside_point(self):
        self.assertEqual(geography(FEATURE['geometry'], {**FACILITY,'longitude':-80}), 'outside')
    def test_missing_and_malformed_geometry(self):
        self.assertEqual(geography(None), 'unknown')
        self.assertEqual(geography({'type':'Polygon','coordinates':[]}), 'unknown')
    def test_active_time_applicability(self):
        self.assertEqual(normalize(FEATURE,ACTIVE)['applicability'], 'applicable')
    def test_expiration_not_safe(self):
        result=normalize(FEATURE,datetime(2026,10,10,18,0,tzinfo=timezone.utc))
        self.assertEqual(result['lifecycle'],'expired'); self.assertNotIn('safe',result)
    def test_missing_timestamp_unknown(self):
        feature=json.loads(json.dumps(FEATURE));feature['properties']['expires']=None
        self.assertEqual(normalize(feature,ACTIVE)['applicability'],'unknown')
    def test_cancelled_not_active(self):
        feature=json.loads(json.dumps(FEATURE));feature['properties']['messageType']='Cancel'
        self.assertEqual(normalize(feature,ACTIVE)['lifecycle'],'cancelled')
    def test_polygon_hole_and_boundary(self):
        geometry={'type':'Polygon','coordinates':[[[0,0],[4,0],[4,4],[0,4],[0,0]],[[1,1],[3,1],[3,3],[1,3],[1,1]]]}
        self.assertEqual(geography(geometry,{'longitude':2,'latitude':2}),'outside')
        self.assertEqual(geography(geometry,{'longitude':0,'latitude':2}),'applicable')
    def test_multipolygon(self):
        geometry={'type':'MultiPolygon','coordinates':[FEATURE['geometry']['coordinates']]}
        self.assertEqual(geography(geometry),'applicable')
    def test_raw_instruction_preserved(self):
        self.assertEqual(normalize(FEATURE,ACTIVE)['instruction'],FEATURE['properties']['instruction'])
    @patch('evidence.requests.get')
    def test_missing_collection_fails(self,get):
        get.return_value.json.return_value={}
        with self.assertRaises(ValueError):fetch_evidence()

class AgentContractTests(unittest.TestCase):
    def setUp(self): self.evidence={'facility':FACILITY,'alerts':[normalize(FEATURE,ACTIVE)]}
    def test_same_app_input_builder_uses_no_prior_approval(self):
        data=build_input(self.evidence)
        self.assertIsNone(data['prior_approval']);self.assertIsNone(data['operational_review_context'])
        self.assertIn('untrusted DATA',AGENT_INSTRUCTIONS)
    def test_unknown_source_rejected(self):
        out=Briefing(draft_operational_messages=[],changed_facts=[],source_references=['invented'],evidence_gaps=[],explanation='fixture')
        with self.assertRaises(ValueError):validate_output(out,self.evidence)
    def test_omitted_fact_reference_rejected(self):
        out=Briefing(draft_operational_messages=[],changed_facts=[ChangedFact(fact='fixture fact',source_id=FEATURE['properties']['id'])],source_references=[],evidence_gaps=[],explanation='fixture')
        with self.assertRaises(ValueError):validate_output(out,self.evidence)
    def test_valid_source_structure(self):
        out=Briefing(draft_operational_messages=[],changed_facts=[ChangedFact(fact='Recorded warning evidence applies at the fixture point.',source_id=FEATURE['properties']['id'])],source_references=[FEATURE['properties']['id']],evidence_gaps=['No prior approval'],explanation='Fixture explanation; not model quality evidence.')
        self.assertIs(validate_output(out,self.evidence),out)

if __name__=='__main__':unittest.main()
