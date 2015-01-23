
ALTER TABLE clinical_pathway ADD COLUMN details_json BLOB;

-- Seed the acne pathway details
UPDATE clinical_pathway SET details_json = '{\"what_is_included\":[\"Response from your doctor within 24 hours\",\"A personalized treatment plan\",\"30 days of follow-up messaging\"],\"who_will_treat_me\":\"Top board-certified dermatologists from across the U.S.\",\"right_for_me\":\"Common acne symptoms include whiteheads, blackheads, and red, inflamed patches of skin (such as cysts).\",\"did_you_know\":[\"You don’t need to let acne run its course. According to the American Academy of Dermatology, 99% of acne cases are treatable.\",\"Prescription treatments can eliminate 84-90% of acne lesions, while non-prescription treatments may eliminate only 16-34%.\",\"Acne is not a teenage problem. 50% of women in their 20s struggle with acne.\",\"95% of patients on Spruce saw substantial improvement in their skin within 12 weeks of their first acne visit.\"],\"faq\":[{\"question\":\"Do I have acne?\",\"answer\":\"Symptoms include whiteheads, blackheads, and red, inflamed patches of skin (such as cysts).\"},{\"question\":\"What if my skin is better today than it normally is?\",\"answer\":\"You’ll be able to tell your dermatologist if your skin is better or worse than normal and you can always message them with additional photos later.\"},{\"question\":\"Will I get a prescription?\",\"answer\":\"Your dermatologist will treat your case in a medically appropriate way, which may include prescriptions. However, prescriptions are not guaranteed, and if you are explicitly looking for Isotretinoin (accutane), you should see a dermatologist in person.\"},{\"question\":\"What if the doctor can\'t treat me?\",\"answer\":\"If your acne cannot be treated remotely, you may be referred to a dermatologist for an in-person appointment. The cost of your visit will be refunded.\"}]}' WHERE id = 1;

-- Update menu for acne for new structure
BEGIN;
UPDATE clinical_pathway_menu SET status = 'INACTIVE';
INSERT INTO clinical_pathway_menu (status, created, json)
    VALUES ('ACTIVE', NOW(), '{"title": "What are you here to see the doctor for today?", "items": [{"title": "Acne", "type": "pathway", "pathway_tag": "health_condition_acne"}]}');
COMMIT;
