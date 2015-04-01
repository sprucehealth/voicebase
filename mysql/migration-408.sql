-- These updates are to ensure that these drugs continue to work in production. They map to the same drug, just the display name has changed.

UPDATE dr_favorite_treatment
SET drug_internal_name = 'Doxycycline (oral - capsule)', dosage_strength = '100 mg'
WHERE drug_internal_name = 'Doxycycline (oral - capsule)'
AND dosage_strength = 'hyclate 100 mg';

UPDATE dr_favorite_treatment
SET drug_internal_name = 'Monodox (oral - capsule)', dosage_strength = '100 mg'
WHERE drug_internal_name = 'Monodox (oral - capsule)'
AND dosage_strength = 'monohydrate 100 mg';

UPDATE dr_favorite_treatment
SET drug_internal_name = 'Doxycycline Monohydrate (oral - tablet)', dosage_strength = 'monohydrate 100 mg'
WHERE drug_internal_name = 'Doxycycline (oral - tablet)'
AND dosage_strength = 'monohydrate 100 mg';

UPDATE dr_favorite_treatment
SET drug_internal_name = 'Doxycycline Monohydrate (oral - capsule)', dosage_strength = 'monohydrate 100 mg'
WHERE drug_internal_name = 'Doxycycline (oral - capsule)'
AND dosage_strength = 'monohydrate 100 mg';

UPDATE dr_favorite_treatment
SET drug_internal_name = 'BenzaClin (topical - gel)', dosage_strength = '1%-5%'
WHERE drug_internal_name = 'BenzaClin (topical - gel)'
AND dosage_strength = '5%-1%';

UPDATE dr_favorite_treatment
SET drug_internal_name = 'Rosanil Cleanser (topical - kit)', dosage_strength = '10%-5%'
WHERE drug_internal_name = 'Rosanil Cleanser (topical - kit)'
AND dosage_strength = '10%-5% with emollients';

UPDATE dr_favorite_treatment
SET drug_internal_name = 'Doxycycline Monohydrate (oral - capsule)', dosage_strength = 'monohydrate 50 mg'
WHERE drug_internal_name = 'Doxycycline (oral - capsule)'
AND dosage_strength = 'monohydrate 50 mg';

UPDATE dr_favorite_treatment
SET drug_internal_name = 'Doxycycline (oral - capsule)', dosage_strength = '150 mg'
WHERE drug_internal_name = 'Doxycycline Monohydrate (oral - capsule)'
AND dosage_strength = 'monohydrate 150 mg';

UPDATE dr_treatment_template
SET drug_internal_name = 'Doxycycline (oral - capsule)', dosage_strength = '100 mg'
WHERE drug_internal_name = 'Doxycycline (oral - capsule)'
AND dosage_strength = 'hyclate 100 mg';

UPDATE dr_treatment_template
SET drug_internal_name = 'Doxycycline (oral - tablet)', dosage_strength = '100 mg'
WHERE drug_internal_name = 'Doxycycline (oral - tablet)'
AND dosage_strength = 'hyclate 100 mg';

UPDATE dr_treatment_template
SET drug_internal_name = 'Doxycycline Monohydrate (oral - tablet)', dosage_strength = 'monohydrate 100 mg'
WHERE drug_internal_name = 'Doxycycline (oral - tablet)'
AND dosage_strength = 'monohydrate 100 mg';

