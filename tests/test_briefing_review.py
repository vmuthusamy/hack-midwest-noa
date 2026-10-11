"""Shared application contract checks; no model invocation or quality claims."""
import copy
import sys
import unittest
from pathlib import Path
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from briefing import Briefing,ChangedFact,DraftMessage,build_input,validate_output,response_schema

class ReplayBriefingContractTests(unittest.TestCase):
    def setUp(self):
        self.evidence={"mode":"REPLAY","facility":{"id":"SC-DEMO-01","fictional":True},"lastSuccessfulRefresh":"2026-10-10T21:07:32Z","alerts":[{"id":"recorded-source-1","event":"Tornado Warning","applicability":"applicable","geography":"applicable","lifecycle":"active"}],"prior_approval":{"instruction":"Report at 10:00","recipient_ids":["S01","S02"]},"recipients":[{"id":"S01","group":"onsite","original_notice_state":"delivered"},{"id":"S02","group":"arriving","original_notice_state":"queued"},{"id":"S03","group":"remote","original_notice_state":"not targeted"}],"review_policy":{"version":"illustrative-v1","eligible":True},"replay_condition":"applicable"}
    def output(self,drafts=None):
        return Briefing(changed_facts=[ChangedFact(fact="Recorded warning applies in this replay.",source_id="recorded-source-1")],source_references=["recorded-source-1"],evidence_gaps=["Synthetic recipients; recorded evidence."],explanation="The earlier arrival instruction requires review in this replay.",draft_operational_messages=drafts if drafts is not None else [DraftMessage(group="onsite",text="The earlier Report at 10:00 instruction is superseded. Onsite reporting is under review; no direction to leave."),DraftMessage(group="arriving",text="The earlier Report at 10:00 instruction is superseded. Arrival is paused pending a manager update.")])
    def test_application_input_has_prior_notice_context(self):
        value=build_input(self.evidence)
        self.assertEqual(value["prior_approval"]["instruction"],"Report at 10:00")
        self.assertEqual(value["synthetic_recipient_groups"],self.evidence["recipients"])
        self.assertTrue(value["operational_review_context"]["eligible"])
        self.assertEqual(value["operational_review_context"]["required_groups"],["onsite","arriving"])
    def test_provider_schema_requires_evidence_and_exact_group_count(self):
        schema=response_schema(self.evidence)
        self.assertEqual(schema['properties']['changed_facts']['minItems'],1)
        self.assertEqual(schema['properties']['source_references']['items']['enum'],['recorded-source-1'])
        self.assertEqual(schema['properties']['draft_operational_messages']['minItems'],2)
        self.assertEqual(schema['properties']['draft_operational_messages']['maxItems'],2)
        self.assertFalse(schema['additionalProperties'])
        blocked=response_schema({**self.evidence,'review_policy':{'eligible':False}})
        self.assertEqual(blocked['properties']['draft_operational_messages']['maxItems'],0)
    def test_exact_affected_groups_and_negative_leave_statement_allowed(self):
        output=self.output();self.assertIs(validate_output(output,self.evidence),output)
    def test_missing_duplicate_or_remote_group_rejected(self):
        for groups in [["onsite"],["onsite","onsite"],["onsite","arriving","remote"]]:
            with self.subTest(groups=groups),self.assertRaises(ValueError):validate_output(self.output([DraftMessage(group=g,text="Arrival under review.") for g in groups]),self.evidence)
    def test_live_has_no_operational_drafts(self):
        evidence={**self.evidence,"mode":"LIVE","prior_approval":None,"recipients":[],"review_policy":{"eligible":False}}
        self.assertIsNone(build_input(evidence)["operational_review_context"])
        with self.assertRaises(ValueError):validate_output(self.output(),evidence)
        self.assertIsInstance(validate_output(self.output([]),evidence),Briefing)
    def test_missing_prior_or_ineligible_policy_has_no_drafts(self):
        for evidence in [{**self.evidence,"prior_approval":None},{**self.evidence,"review_policy":{"eligible":False}}]:
            with self.subTest(evidence=evidence),self.assertRaises(ValueError):validate_output(self.output(),evidence)
            self.assertIsInstance(validate_output(self.output([]),evidence),Briefing)
    def test_unsupported_safety_directives_rejected(self):
        for text in ["Evacuate the facility now.","Leave the building immediately.","Do not leave your location.","Go home now.","Drive to the community center.","Shelter in Room 12.","It is safe to reopen.","Guaranteed delivery to every employee."]:
            with self.subTest(text=text),self.assertRaises(ValueError):validate_output(self.output([DraftMessage(group="onsite",text="The earlier Report at 10:00 instruction is superseded. "+text),DraftMessage(group="arriving",text="The earlier Report at 10:00 instruction is superseded. Arrival is paused pending manager update.")]),self.evidence)
    def test_applicable_alert_fact_cannot_be_omitted(self):
        output=self.output();output.changed_facts=[]
        with self.assertRaises(ValueError):validate_output(output,self.evidence)
    def test_missing_evidence_cannot_reauthorize_prior_instruction(self):
        evidence={**self.evidence,'review_policy':{'eligible':False}}
        for text in ['The prior Report at 10:00 instruction stands unchanged.','The instruction remains in effect.','Release the held notices.','Resume staff arrivals.']:
            with self.subTest(text=text),self.assertRaises(ValueError):
                output=self.output([]);output.explanation=text;validate_output(output,evidence)
    def test_cancelled_warning_cannot_be_described_as_expired(self):
        evidence=copy.deepcopy(self.evidence)
        evidence["alerts"][0].update(lifecycle="cancelled",applicability="not-applicable",messageType="Cancel")
        evidence["review_policy"]["eligible"]=False
        output=self.output([])
        output.explanation="The recorded warning is cancelled and not applicable."
        output.changed_facts=[ChangedFact(fact="The recorded warning expired at 5:15 PM EDT.",source_id="recorded-source-1"),ChangedFact(fact="The warning is marked cancelled.",source_id="recorded-source-1")]
        with self.assertRaises(ValueError):validate_output(output,evidence)
        output.changed_facts[0].fact="The recorded warning was cancelled before its scheduled expiry at 5:15 PM EDT."
        self.assertIs(validate_output(output,evidence),output)
    def test_material_update_must_appear_in_evidence_summary(self):
        evidence=copy.deepcopy(self.evidence);evidence["alerts"][0]["messageType"]="Update"
        output=self.output()
        # A recipient's 'pending manager update' is not an alert-update explanation.
        with self.assertRaises(ValueError):validate_output(output,evidence)
        output.changed_facts[0].fact="The source marks this recorded warning as an updated alert."
        self.assertIs(validate_output(output,evidence),output)
    def test_tool_markup_and_unmatched_quotes_rejected(self):
        for text in ['Recorded warning.</explanation><parameter name="drafts">[]','Recorded "warning.','Recorded “warning.']:
            with self.subTest(text=text),self.assertRaises(ValueError):
                output=self.output();output.explanation=text;validate_output(output,self.evidence)
    def test_recipient_text_cannot_assert_workflow_state(self):
        for text in ['This is a simulated intent.','This is not an approved message.','This draft is queued.','This message was sent.']:
            with self.subTest(text=text),self.assertRaises(ValueError):
                output=self.output();output.draft_operational_messages[0].text+=' '+text;validate_output(output,self.evidence)
    def test_lone_trailing_provider_quote_is_normalized(self):
        output=self.output();output.explanation='Recorded warning."'
        self.assertEqual(validate_output(output,self.evidence).explanation,'Recorded warning.')
    def test_references_must_exist_and_changed_facts_must_be_cited(self):
        output=self.output();output.source_references=["invented"]
        with self.assertRaises(ValueError):validate_output(output,self.evidence)
        output=self.output();output.source_references=[]
        with self.assertRaises(ValueError):validate_output(output,self.evidence)

if __name__=="__main__":unittest.main()
