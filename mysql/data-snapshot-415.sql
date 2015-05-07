-- MySQL dump 10.13  Distrib 5.6.24, for osx10.10 (x86_64)
--
-- Host: localhost    Database: database_9661
-- ------------------------------------------------------
-- Server version	5.6.24

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `app_text`
--

DROP TABLE IF EXISTS `app_text`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `app_text` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `comment` varchar(600) DEFAULT NULL,
  `app_text_tag` varchar(250) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `app_text_tag` (`app_text_tag`)
) ENGINE=InnoDB AUTO_INCREMENT=512 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_text`
--

LOCK TABLES `app_text` WRITE;
/*!40000 ALTER TABLE `app_text` DISABLE KEYS */;
INSERT INTO `app_text` VALUES (1,'reason for visit with doctor','txt_visit_reason'),(2,'acne is the reason for visit','txt_acne_visit_reason'),(3,'something else is reason for visit','txt_something_else_visit_reason'),(4,'hint for typing a symptom or condition','txt_hint_type_symptom'),(5,'duration of acne','txt_acne_length'),(6,'0-6 months for acne length','txt_less_six_months'),(7,'6-12 months for acne length','txt_six_months_one_year_acne_length'),(8,'1-2 years for acne length','txt_one_two_year_acne_length'),(9,'2+ years for acne length','txt_two_plus_year_acne_length'),(10,'is your acne getting worse','txt_acne_worse'),(11,'acne is getting worse response','txt_yes'),(12,'acne is not getting worse response','txt_no'),(13,'helper text to describe what is making acne worse','txt_describe_changes_acne_worse'),(14,'hint text giving examples for what makes acne worse','txt_examples_changes_acne_worse'),(15,'select type of treatments tried for acne','txt_acne_treatments'),(16,'over the counter treatment for acne','txt_otc_acne_treatment'),(17,'prescription treatment for acne','txt_prescription_treatment'),(18,'no treatment tried for acne','txt_no_treatment_acne'),(19,'list medications tried for acne','txt_list_medications_acne'),(20,'type to add treatment','txt_type_add_treatment'),(21,'share anything else w.r.t acne','txt_anything_else_acne'),(22,'hint for anything else you\'d like to tell the doctor','txt_hint_anything_else_acne_treatment'),(23,'question for females to learn about family planning','txt_pregnancy_planning'),(26,'Are you allergic to any medications?','txt_allergic_to_medications'),(29,'hint to add a medication','txt_type_add_medication'),(30,'Your Skin History','txt_skin_history'),(31,'Your Medical History','txt_medical_history'),(32,'question to list medications','txt_list_medications'),(33,'hint to list medications','txt_hint_list_medications'),(34,'question to get social history','txt_get_social_history'),(35,'smoke tobacco','txt_smoke_social_history'),(36,'drink alocohol','txt_alcohol_social_history'),(37,'use tanning beds','txt_tanning_social_history'),(38,'question to learn whether patient has been diagnosed in the past','txt_diagnosed_skin_past'),(39,'listing past skin diagnosis for paitent to chose from','txt_alopecia_diagnosis'),(40,'listing past sking diagnoses for patient to chose from','txt_acne_diagnosis'),(41,'listing past sking diagnoses for patient to chose from','txt_eczema'),(42,'listing past sking diagnoses for patient to chose from','txt_psoriasis_diagnosis'),(43,'listing past sking diagnoses for patient to chose from','txt_rosacea_diagnosis'),(44,'listing past sking diagnoses for patient to chose from','txt_skin_cancer_diagnosis'),(45,'listing past sking diagnoses for patient to chose from','txt_other_diagnosis'),(46,'question to list any medical conditions that patient has been treated for','txt_list_medical_condition'),(47,'hint to prompt user to add a condition','txt_hint_add_condition'),(48,'medical condition list to chose from','txt_arthritis_condition'),(49,'medical condition list to chose from','txt_artificial_heart_valve_condition'),(50,'medical condition list to chose from','txt_artificial_joint_condition'),(51,'medical condition list to chose from','txt_asthma_condition'),(52,'medical condition list to chose from','txt_blood_clots_condition'),(53,'medical condition list to chose from','txt_diabetes_condition'),(54,'medical condition list to chose from','txt_epilepsy_condition'),(55,'medical condition list to chose from','txt_high_bp_condition'),(56,'medical condition list to chose from','txt_high_cholestrol_condition'),(57,'medical condition list to chose from','txt_hiv_condition'),(58,'medical condition list to chose from','txt_heart_attack_condition'),(59,'medical condition list to chose from','txt_heart_murmur_condition'),(60,'medical condition list to chose from','txt_irregular_heartbeat_condition'),(61,'medical condition list to chose from','txt_kidney_disease_condition'),(62,'medical condition list to chose from','txt_liver_disease_condition'),(63,'medical condition list to chose from','txt_lung_disease_condition'),(64,'medical condition list to chose from','txt_lupus_disease_condition'),(65,'medical condition list to chose from','txt_organ_transplant_disease_condition'),(66,'medical condition list to chose from','txt_pacemaker_disease_condition'),(67,'medical condition list to chose from','txt_thyroid_problems_condition'),(68,'medical condition list to chose from','txt_other_condition_condition'),(69,'medical condition list to chose from','txt_no_condition'),(70,'question to determine where the patient is experiencing acne','txt_acne_location'),(71,'face location for acne','txt_face_acne_location'),(72,'chest location for acne','txt_chest_acne_location'),(73,'back location for acne','txt_back_acne_location'),(74,'other locations for acne','txt_other_acne_location'),(75,'title for face section of photo tips','txt_face_photo_tips_title'),(76,'description for face section of photo taking','txt_photo_tips_description'),(77,'tip to remove glasses','txt_remove_glasses_tip'),(78,'tip to pull hair back','txt_pull_hair_back_tip'),(79,'tip to have no makeup','txt_no_makeup_tip'),(80,'title for chest section photo tips','txt_chest_photo_tips_title'),(81,'tip to remove jewellery','txt_remove_jewellery_photo_tip'),(82,'face front label','txt_face_front'),(83,'profile left label','txt_profile_left'),(84,'profile right label','txt_profile_right'),(85,'chest label','txt_chest'),(86,'back lebel','txt_back'),(87,'title for photo section','txt_photo_section_title'),(88,'short description of reason for visit','txt_short_reason_visit'),(89,'short description for length of time patient has been experiencing acne','txt_short_acne_length'),(90,'short description of other symptoms that the patient is attempting to use the app for ','txt_short_other_symptoms'),(91,'short description of whether or not acne is getting worse','txt_short_acne_worse'),(92,'short description of changes that would be making acne worse','txt_short_changes_acne_worse'),(93,'short description of previous types of treatments tried','txt_short_prev_type_treatment'),(94,'short description of previous list of treatments that have been tried','txt_short_prev_list_treatment'),(95,'short description of anything else patient would like to tell doctor about cane','txt_short_anything_else_acne'),(96,'short description of all the places that the patient marked acne is being present on','txt_short_photo_locations'),(97,'short description of whether patient is planning pregnancy','txt_short_pregnant'),(98,'short description of whether patient is alergic to medications','txt_allergic_medications'),(99,'short description to list any medications patient is currently taking','txt_short_list_medications'),(100,'short description to describe social history of patient','txt_short_social_history'),(101,'short description for previous skin diagnosis','txt_short_prev_skin_diagnosis'),(102,'short description for patient to describe medical conditions that they have been treated for','txt_short_medical_condition'),(103,'prompt to take photo of treatment','txt_take_photo_treatment'),(104,'short description for a list of medications that patient is allergic to','txt_short_allergic_medications_list'),(105,'short description for front face photo of patient','txt_short_face_photo'),(106,'short description for chest photos of patient','txt_short_chest_photo'),(107,'short description for back photo of patient','txt_short_back_photo'),(108,'short description for other photo of patient','txt_short_other_photo'),(109,'other lable for photo taking','txt_other'),(110,'how effective was this treatment','txt_effective_treatment'),(111,'answer option','txt_not_very'),(112,'answer option','txt_somewhat'),(113,'answer option','txt_very'),(114,'are you currently using this treatment','txt_current_treatment'),(115,'less than 1 month','txt_one_or_less'),(116,'2-5 months','txt_two_five_months'),(117,'6-11 months','txt_six_eleven_months'),(118,'12+ months','txt_twelve_plus_months'),(119,'not very effective','txt_not_very_effective'),(120,'somewhat effective','txt_somewhat_effective'),(121,'very effective','txt_very_effective'),(122,'currently using it','txt_current_using'),(123,'not currently using it','txt_not_currently_using'),(124,'Used for less than 1 month','txt_used_less_1_month'),(125,'Used for 2-5 months','txt_used_two_five_months'),(126,'Used for 6-11 months','txt_used_six_eleven_months'),(127,'Used for over a year','txt_used_twelve_plus_months'),(128,'question for length of treatment','txt_treatment_length'),(150,'txt for when you first started experiencing acne','txt_first_acne_experience'),(151,'txt response of during puberty','txt_during_puberty'),(152,'txt response of within last six months','txt_within_last_six_months'),(153,'txt response of 1-2 years ago','txt_one_two_years_ago'),(154,'txt response of more than 2 years ago','txt_more_than_two_years'),(155,'txt summary for onset of symptoms','txt_onset_symptoms'),(156,'txt for asking the user if they are experiencing acne symptoms','txt_acne_symtpoms'),(157,'txt for response of acne being painful to touch','txt_painful_touch'),(158,'txt for response of acne being scarring','txt_scarring'),(159,'txt for response of acne causing discoloration','txt_discoloration'),(160,'txt for summarizing additional symptoms','txt_additional_symptoms'),(161,'txt for asking female patients if their acne gets worse with periods','txt_acne_worse_period'),(162,'txt for asking female patients if their periods are regular','txt_periods_regular'),(163,'txt for summarizing information about txt_menstrual_cycle','txt_menstrual_cycle'),(164,'txt for question to descibe skin','txt_skin_description'),(165,'txt for response to skin description as normal','txt_normal_skin'),(166,'txt for response to skin description as oily','txt_oily_skin'),(167,'txt for response to skin description as dry','txt_dry_skin'),(168,'txt for response to skin description as combination','txt_combination_skin'),(169,'txt for summarizing skin type','txt_skin_type'),(170,'txt for determining whether patient has been allergic to topical medication','txt_allergy_topical_medication'),(171,'txt summary for determining whether patient has been allergic to topical medication','txt_summary_allergy_topical_medication'),(172,'txt for determining any other conditions patient may have been diagnosed for in the past','txt_other_condition_acne'),(173,'txt for determining any other conditions patient may have been diagnosed for in the past','txt_summary_other_condition_acne'),(174,'txt response for determining any other conditions patient may have been diagnosed for in the past','txt_gasitris'),(175,'txt response for determining any other conditions patient may have been diagnosed for in the past','txt_colitis'),(176,'txt response for determining any other conditions patient may have been diagnosed for in the past','txt_kidney_disease'),(177,'txt response for determining any other conditions patient may have been diagnosed for in the past','txt_lupus'),(178,'txt summary for treatment not effective','txt_answer_summary_not_effective'),(179,'txt summary for treatment somewhat effective','txt_answer_summary_somewhat_effective'),(180,'txt summary for treatment very effective','txt_answer_summary_very_effective'),(181,'txt summary for not currently using treatment','txt_answer_summary_not_using'),(182,'txt summary for using current treatment','txt_answer_summary_using'),(183,'txt summary for using treatment less than a month','txt_answer_summary_less_month'),(184,'txt summary for using treatment 2-5 months','txt_answer_summary_two_five_months'),(185,'txt summary for using treamtent 6-11 months','txt_answer_summary_six_eleven_months'),(186,'txt summary for using treatment 12+ months','txt_answer_summary_twelve_plus_months'),(187,'txt for prompting user to add treatment','txt_add_treatment'),(188,'txt for prompting user to add medication','txt_add_medication'),(189,'txt for prompting user to take a photo of the medication','txt_take_photo_medication'),(190,'txt for button when adding medication','txt_add_button_medication'),(191,'txt for button when adding treatment','txt_add_button_treatment'),(192,'txt for saving changes when adding medication or treatment','txt_save_changes'),(193,'txt for button to remove treatment','txt_remove_treatment'),(194,'txt for button to remove medication','txt_remove_medication'),(195,'what is your diagnosisa','txt_what_diagnosis'),(196,'acne vulgaris','txt_acne_vulgaris'),(197,'acne rosacea','txt_acne_rosacea'),(198,'how severe is the patients acne','txt_acne_severity'),(199,'acne severity mild','txt_acne_severity_mild'),(200,'acne severity moderate','txt_acne_severity_moderate'),(201,'acne severity severe','txt_acne_severity_severe'),(202,'type of acne','txt_acne_type'),(203,'acne whiteheads','txt_acne_whiteheads'),(204,'acne pustules','txt_acne_pustules'),(205,'acne nodules','txt_acne_nodules'),(206,'acne inflammatory','txt_acne_inflammatory'),(207,'acne blackheads','txt_acne_blackheads'),(208,'acne papules','txt_acne_papules'),(209,'acne cysts','txt_acne_cysts'),(210,'acne hormonal','txt_acne_hormonal'),(211,'select all apply','txt_select_all_apply'),(212,'dispense unit','txt_dispense_unit_Bag'),(213,'dispense unit','txt_dispense_unit_Bottle'),(214,'dispense unit','txt_dispense_unit_Box'),(215,'dispense unit','txt_dispense_unit_Capsule'),(216,'dispense unit','txt_dispense_unit_Cartridge'),(217,'dispense unit','txt_dispense_unit_Container'),(218,'dispense unit','txt_dispense_unit_Drop'),(219,'dispense unit','txt_dispense_unit_Gram'),(220,'dispense unit','txt_dispense_unit_Inhaler'),(221,'dispense unit','txt_dispense_unit_International'),(222,'dispense unit','txt_dispense_unit_Kit'),(223,'dispense unit','txt_dispense_unit_Liter'),(224,'dispense unit','txt_dispense_unit_Lozenge'),(225,'dispense unit','txt_dispense_unit_Milligram'),(226,'dispense unit','txt_dispense_unit_Milliliter'),(227,'dispense unit','txt_dispense_unit_Million_Units'),(228,'dispense unit','txt_dispense_unit_Mutually_Defined'),(229,'dispense unit','txt_dispense_unit_Fluid_Ounce'),(230,'dispense unit','txt_dispense_unit_Not_Specified'),(231,'dispense unit','txt_dispense_unit_Pack'),(232,'dispense unit','txt_dispense_unit_Packet'),(233,'dispense unit','txt_dispense_unit_Pint'),(234,'dispense unit','txt_dispense_unit_Suppository'),(235,'dispense unit','txt_dispense_unit_Syringe'),(236,'dispense unit','txt_dispense_unit_Tablespoon'),(237,'dispense unit','txt_dispense_unit_Tablet'),(238,'dispense unit','txt_dispense_unit_Teaspoon'),(239,'dispense unit','txt_dispense_unit_Transdermal_Patch'),(240,'dispense unit','txt_dispense_unit_Tube'),(241,'dispense unit','txt_dispense_unit_Unit'),(242,'dispense unit','txt_dispense_unit_Vial'),(243,'dispense unit','txt_dispense_unit_Each'),(244,'dispense unit','txt_dispense_unit_Gum'),(245,'dispense unit','txt_dispense_unit_Ampule'),(246,'dispense unit','txt_dispense_unit_Applicator'),(247,'dispense unit','txt_dispense_unit_Applicatorful'),(248,'dispense unit','txt_dispense_unit_Bar'),(249,'dispense unit','txt_dispense_unit_Bead'),(250,'dispense unit','txt_dispense_unit_Blister'),(251,'dispense unit','txt_dispense_unit_Block'),(252,'dispense unit','txt_dispense_unit_Bolus'),(253,'dispense unit','txt_dispense_unit_Can'),(254,'dispense unit','txt_dispense_unit_Canister'),(255,'dispense unit','txt_dispense_unit_Capler'),(256,'dispense unit','txt_dispense_unit_Carton'),(257,'dispense unit','txt_dispense_unit_Case'),(258,'dispense unit','txt_dispense_unit_Cassette'),(259,'dispense unit','txt_dispense_unit_Cylinder'),(260,'dispense unit','txt_dispense_unit_Disk'),(261,'dispense unit','txt_dispense_unit_Dose_Pack'),(262,'dispense unit','txt_dispense_unit_Dual_Packs'),(263,'dispense unit','txt_dispense_unit_Film'),(264,'dispense unit','txt_dispense_unit_Gallon'),(265,'dispense unit','txt_dispense_unit_Implant'),(266,'dispense unit','txt_dispense_unit_Inhalation'),(267,'dispense unit','txt_dispense_unit_Inhaler_Refill'),(268,'dispense unit','txt_dispense_unit_Insert'),(269,'dispense unit','txt_dispense_unit_Intravenous_Bag'),(270,'dispense unit','txt_dispense_unit_Milimeter'),(271,'dispense unit','txt_dispense_unit_Nebule'),(272,'dispense unit','txt_dispense_unit_Needle_Free_Injection'),(273,'dispense unit','txt_dispense_unit_Oscular_System'),(274,'dispense unit','txt_dispense_unit_Ounce'),(275,'dispense unit','txt_dispense_unit_Pad'),(276,'dispense unit','txt_dispense_unit_Paper'),(277,'dispense unit','txt_dispense_unit_Pouch'),(278,'dispense unit','txt_dispense_unit_Pound'),(279,'dispense unit','txt_dispense_unit_Puff'),(280,'dispense unit','txt_dispense_unit_Quart'),(281,'dispense unit','txt_dispense_unit_Ring'),(282,'dispense unit','txt_dispense_unit_Sachet'),(283,'dispense unit','txt_dispense_unit_Scoopful'),(284,'dispense unit','txt_dispense_unit_Sponge'),(285,'dispense unit','txt_dispense_unit_Spray'),(286,'dispense unit','txt_dispense_unit_Stick'),(287,'dispense unit','txt_dispense_unit_Strip'),(288,'dispense unit','txt_dispense_unit_Swab'),(289,'dispense unit','txt_dispense_unit_Tabminder'),(290,'dispense unit','txt_dispense_unit_Tampon'),(291,'dispense unit','txt_dispense_unit_Tray'),(292,'dispense unit','txt_dispense_unit_Troche'),(293,'dispense unit','txt_dispense_unit_Wafer'),(294,'text to explain to customer that we are only diagnosing for acne currently','txt_condition_diagnosis_title'),(295,'placeholder text to explain to customer that we are only diagnosing for acne currently','txt_condition_diagnosis_placeholder'),(296,'Submit','txt_submit'),(297,'cysts option for symptoms','txt_cysts'),(298,'none of the above multiple choice option','txt_none_of_the_above'),(299,'txt for did this treatment irritate your skin','txt_irritate_skin'),(300,'summary text for treatment irritating skin','txt_irritated_skin_summary'),(301,'summary text for treatment irritating skin','txt_not_irritated_skin_summary'),(302,'option to indicate that the patient is pregnant','txt_pregnant'),(303,'option to indicate that the patient is nursing','txt_nursing'),(304,'option to indicate that the patient is planning a pregnancy','txt_planning_pregnancy'),(305,'option to indicate that the patient neither pregnant nor planning a pregnancy','txt_pregnancy_nursing_none'),(306,'question text for number of months medication has been taken for','txt_months_current_medication'),(307,'option to indicate that the patient has taken medication for less than one month','txt_answer_summary_taken_less_one_month'),(308,'option to indicate that the patient has taken medication for 2-5 months','txt_answer_summary_taken_two_five_months'),(309,'option to indicate that the patient has taken medication for 6-11 months','txt_answer_summary_taken_six_eleven_months'),(310,'option to indicate that the patient has taken medication for 12+ months','txt_answer_summary_taken_twelve_plus_months'),(311,'hypertension','txt_hypertension'),(312,'polycystic ovary syndrome','txt_poly_ovary_syndrome'),(313,'select which skin condition','txt_select_skin_condition'),(314,'option for acne location','txt_neck'),(315,'question title for other acne location','txt_other_acne_location_prompt'),(316,'type to add a location','txt_type_add_location'),(317,'type to add a condition','txt_prompt_add_skin_condition'),(318,'regular periods summary','txt_summary_periods_regular'),(319,'worse periods summary','txt_summary_periods_worse'),(320,'potential environment factors','txt_summary_environment_factors'),(321,'is pregnant summary','txt_summary_is_pregnant'),(322,'is nursing summary','txt_summary_is_nursing'),(323,'is planning pregnancy summary','txt_summary_planning_pregnancy'),(324,'is not pregnant planning or nursing summary','txt_summary_not_pregant_planning_nursing'),(325,'summary for other past skin condition','txt_summary_other_past_skin_condition'),(326,'perioral dermatitis','txt_perioral_dermitits'),(327,'comedonal','txt_comedonal'),(328,'Erythematotelangiectatic Rosacea','txt_erythematotelangiectatic_rosacea'),(329,'Papulopustular Rosacea','txt_papulopstular_rosacea'),(330,'Rhinophyma','txt_rhinophyma_rosacea'),(331,'Ocular Rosacea','txt_ocular_rosacea'),(332,'acne','txt_acne'),(333,'6-12 months ago','txt_six_twelve_months_ago'),(334,'are you currently taking any medications','txt_current_medications_yes_no'),(335,'List any other than those you may be using for acne.','txt_list_other_than_acne'),(336,'none','txt_none'),(337,'other location specified','txt_other_location_specified'),(338,'empty state text','txt_empty_state_q_allergic_medication_entry'),(339,'empty state text','txt_empty_state_q_current_medications_entry'),(340,'empty state text','txt_empty_state_q_list_prev_skin_condition_diagnosis'),(341,'empty state text','txt_empty_state_q_changes_acne_worse'),(342,'empty state text','txt_empty_state_q_acne_prev_treatment_list'),(343,'empty state text','txt_empty_state_q_anything_else_acne'),(344,'alert text','txt_alert_q_pregnancy_planning'),(345,'alert text','txt_alert_q_allergic_medication_entry'),(346,NULL,'text_medication_entry_q'),(347,NULL,'txt_prescription_preference_q'),(348,NULL,'txt_generic_only'),(349,NULL,'txt_no_preference'),(350,NULL,'txt_generic_rx_only_alert'),(351,NULL,'txt_prescription_preference_short'),(352,NULL,'text_intestinal_inflammation'),(353,NULL,'text_organ_transplant'),(354,NULL,'text_pregnancy_disclaimer'),(355,NULL,'text_no_pregnancy'),(356,NULL,'txt_picked_or_squeezed'),(357,NULL,'txt_created_scars'),(358,NULL,'txt_acne_prev_prescriptions_q'),(359,NULL,'txt_tried_otc_acne'),(360,NULL,'txt_list_otc_products'),(361,NULL,'txt_placeholder_anything_else_acne'),(362,NULL,'txt_using_otc_q'),(363,NULL,'txt_effective_otc_q'),(364,NULL,'txt_otc_irritate_skin_q'),(365,NULL,'txt_length_otc_q'),(366,NULL,'txt_add_product'),(367,NULL,'txt_remove_product'),(368,NULL,'txt_type_to_add_product'),(369,NULL,'txt_empty_state_q_acne_prev_otc_list'),(371,NULL,'txt_otc_tried'),(372,NULL,'txt_is_pregnant'),(373,NULL,'txt_not_pregnant'),(374,NULL,'txt_not_suitable_spruce'),(375,NULL,'txt_describe_patient_condition'),(376,NULL,'txt_type_diagnosis'),(377,NULL,'txt_why_visit_not_suitable_spruce'),(378,NULL,'txt_for_internal_purposes'),(379,NULL,'txt_describe_why_not_able_to_treat'),(380,NULL,'txt_other_location'),(381,NULL,'txt_prev_prescriptions_select'),(382,NULL,'txt_benzaclin'),(383,NULL,'txt_benzoyl_peroxide'),(384,NULL,'txt_clindamycin'),(385,NULL,'txt_differin'),(386,NULL,'txt_duac'),(387,NULL,'txt_epiduo'),(388,NULL,'txt_metrogel'),(389,NULL,'txt_minocycline'),(390,NULL,'txt_retina_or_tretinoin'),(391,NULL,'txt_tetracycline'),(392,NULL,'txt_currently_using_it'),(393,NULL,'txt_how_effective'),(394,NULL,'txt_not'),(395,NULL,'txt_not_effective'),(396,NULL,'txt_did_you_use_for_more_three_months'),(397,NULL,'txt_used_more_than_three_months'),(398,NULL,'txt_did_not_use_more_than_three_months'),(399,NULL,'txt_did_it_irritate_skin'),(400,NULL,'txt_anything_else_tell_doctor'),(401,NULL,'txt_optional'),(402,NULL,'txt_prev_otc_select'),(403,NULL,'txt_acne_free'),(404,NULL,'txt_cetaphil'),(405,NULL,'txt_clean_and_clear'),(406,NULL,'txt_clearasil'),(407,NULL,'txt_noxzema'),(408,NULL,'txt_oxy'),(409,NULL,'txt_proactiv'),(410,NULL,'txt_zeno'),(411,NULL,'txt_type_another_treatment'),(412,NULL,'txt_formatted_name_product_tried'),(413,NULL,'txt_currently_using'),(414,NULL,'txt_effective'),(415,NULL,'txt_used_three_plus_months'),(416,NULL,'txt_irritating'),(417,NULL,'txt_comments'),(418,NULL,'txt_which_product'),(419,NULL,'txt_how_long'),(420,NULL,'txt_face_side'),(421,NULL,'txt_aveeno'),(422,NULL,'txt_panoyl'),(423,NULL,'txt_doxycycline'),(424,'text for how skin compares based on photos','txt_skin_photo_comparison'),(425,'text for how skin compares based on photos','txt_more_acne_blemishes'),(426,'text for how skin compares based on photos','txt_summary_more_acne_blemishes'),(427,'text for how skin compares based on photos','txt_fewer_acne_blemishes'),(428,'text for how skin compares based on photos','txt_summary_fewer_acne_blemishes'),(429,'text for how skin compares based on photos','txt_about_the_same'),(430,'text for how skin compares based on photos','txt_summary_about_the_same'),(431,'text for how skin compares based on photos','txt_short_skin_photo_comparison'),(432,'text insurance coverage info','txt_insurance_coverage'),(433,'text insurance coverage info','txt_insurance_brand_generic'),(434,'text insurance coverage info','txt_insurance_generic_only'),(435,'text insurance coverage info','txt_insurance_idk'),(436,'text insurance coverage info','txt_no_insurance'),(437,'text insurance coverage info','txt_short_insurance_coverage'),(438,'text insurance coverage info','txt_summary_no_insurance'),(439,'text insurance coverage info','txt_summary_insurance_idk'),(440,'text skin description','txt_sensitive_skin_option'),(441,'placeholder text for adding another skin description','txt_type_another_description'),(442,'acne symptoms nodules','txt_deep_lumps'),(443,'text if acne has been made worse by something','txt_acne_worse_by_something'),(444,'text if acne has been made worse by something','txt_short_acne_worse_by_something'),(445,'options for why acne may be worse','txt_diet'),(446,'options for why acne may be worse','txt_hair_products'),(447,'options for why acne may be worse','txt_makeup'),(448,'options for why acne may be worse','txt_hormonal_changes'),(449,'options for why acne may be worse','txt_stress'),(450,'options for why acne may be worse','txt_sweating_and_sports'),(451,'options for why acne may be worse','txt_weather'),(452,'options for why acne may be worse','txt_none_or_not_sure'),(453,'options for why acne may be worse','txt_short_none_or_not_sure'),(454,'placeholder text for adding another contributing factor','txt_type_another_factor'),(455,'option for skin description','txt_neutrogena'),(456,'text for other placeholder text to add another condition','txt_add_condition'),(457,'text for how happy patient is with improvements to skin','txt_how_happy'),(458,'text for how happy patient is with improvements to skin','txt_how_happy_short'),(459,'text for how happy patient is with improvements to skin','txt_very_happy'),(460,'text for how happy patient is with improvements to skin','txt_happy'),(461,'text for how happy patient is with improvements to skin','txt_neutral'),(462,'text for how happy patient is with improvements to skin','txt_unhappy'),(463,'text for how happy patient is with improvements to skin','txt_very_unhappy'),(464,'text for how happy patient is with improvements to skin','txt_why_less_than_happy'),(465,'text for how happy patient is with improvements to skin','txt_why_less_than_happy_short'),(466,'Patient chose not to answer','txt_patient_chose_not_to_answer'),(467,'This will help your doctor make any necessary adjustments to your plan.','txt_doctor_make_adjustments'),(468,'text for tp compliance','txt_using_tp_as_instructed'),(469,'text for tp compliance','txt_using_tp_as_instructed_short'),(470,'text for tp compliance','txt_tp_compliance_yes'),(471,'text for tp compliance','txt_mostly'),(472,'text for tp compliance','txt_im_not_sure'),(473,'text for tp compliance','txt_compliant'),(474,'text for tp compliance','txt_mostly_compliant'),(475,'text for tp compliance','txt_somewhat_compliant'),(476,'text for tp compliance','txt_not_compliant'),(477,'text for tp compliance','txt_not_sure'),(478,'text side effects from medications','txt_side_effects'),(479,'text side effects from medications','txt_side_effects_short'),(480,'text side effects from medications','txt_side_effects_explain'),(481,'text side effects from medications','txt_description'),(482,'using all treatments in plan?','txt_using_all_treatments_in_plan'),(483,'using all treatments in plan?','txt_using_all_treatments_in_plan_short'),(484,'treamtents in tp that patient stopped using and why','txt_treatments_in_tp_stopped_using'),(485,'treamtents in tp that patient stopped using and why','txt_treatments_in_tp_stopped_using_short'),(486,'difficulty in complying with treatment plan','txt_tp_compliance_difficulty'),(487,'difficulty in complying with treatment plan','txt_tp_compliance_difficulty_short'),(488,'medications other than prescribed for acne since tp','txt_other_medications_since_tp'),(489,'medications other than prescribed for acne since tp','txt_other_medications_since_tp_entry'),(490,'current medications','txt_current_medications'),(491,'no medications specified','txt_no_medications_specified'),(492,'placeholder text for helping patient use treatment plan more effectively','txt_questions_tp_effectively'),(493,'medication allergies since last visit','txt_medication_allergies_since_visit'),(494,'changes to medical history that may be relevant','txt_med_hx_changes_relevance'),(495,'changes to medical history that may be relevant','txt_med_hx_changes_relevance_short'),(496,'treatment plan','txt_treatment_plan'),(497,'text side effects from medications','txt_med_hx_describe_changes'),(498,'text side effects from medications','txt_med_hx_describe_changes_short'),(499,'placeholder text','txt_placeholder_tp_difficulty'),(500,'sometimes','txt_sometimes'),(501,NULL,'txt_severity'),(502,NULL,'txt_mild'),(503,NULL,'txt_moderate'),(504,NULL,'txt_severe'),(505,NULL,'txt_type'),(506,NULL,'txt_cystic'),(507,NULL,'txt_hormonal'),(508,'In-app feedback screen title','txt_feedback_screen_title'),(509,'In-app feedback rating prompt','txt_feedback_rating_prompt'),(510,'In-app feedback comment placeholder','txt_feedback_comment_placeholder'),(511,'In-app feedback submit button','txt_feedback_submit_button');
/*!40000 ALTER TABLE `app_text` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `localized_text`
--

DROP TABLE IF EXISTS `localized_text`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `localized_text` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `language_id` int(10) unsigned NOT NULL,
  `ltext` varchar(600) NOT NULL,
  `app_text_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `language_id` (`language_id`,`app_text_id`),
  KEY `app_text_id` (`app_text_id`),
  CONSTRAINT `localized_text_ibfk_1` FOREIGN KEY (`app_text_id`) REFERENCES `app_text` (`id`) ON DELETE CASCADE,
  CONSTRAINT `localized_text_ibfk_2` FOREIGN KEY (`language_id`) REFERENCES `languages_supported` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=516 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `localized_text`
--

LOCK TABLES `localized_text` WRITE;
/*!40000 ALTER TABLE `localized_text` DISABLE KEYS */;
INSERT INTO `localized_text` VALUES (4,1,'What\'s the reason for your visit with Dr. %s today?',1),(5,1,'Other',3),(6,1,'Acne',2),(7,1,'Type a symptom or condition',4),(8,1,'How long have you been experiencing acne symptoms?',5),(9,1,'0-6 months',6),(10,1,'6-123 months',7),(11,1,'1-2 years',8),(12,1,'2+ years',9),(13,1,'Has your acne been getting worse?',10),(14,1,'Yes',11),(15,1,'No',12),(16,1,'Are there any recent changes that could be affecting your skin?',13),(18,1,'Ex: new cosmetics, sports, warmer weather, increased stress.',14),(19,1,'What types of treatments have you previously tried for your acne?',15),(20,1,'Over the counter',16),(21,1,'Prescription',17),(22,1,'No treatments tried',18),(23,1,'List medications that you are currently using or have tried in the past.',19),(24,1,'Type to add a medication',20),(25,1,'Is there anything else you\'d like to share about your skin?',21),(26,1,'Anything else you’d like your doctor to know?',22),(27,1,'Are you pregnant, planning a pregnancy or nursing?',23),(28,1,'Are you allergic to any medications?',26),(29,1,'Type to add a medication',29),(30,1,'About Your Skin',30),(31,1,'Medical History',31),(32,1,'Which medications are you currently taking?',32),(33,1,'Include birth control, over the counter medications, vitamins or herbal supplements that you may be currently taking.',33),(34,1,'Select which if any of the following activities you do regularly:',34),(35,1,'Smoke tobacco',35),(36,1,'Drink alcohol',36),(37,1,'Use tanning beds or sunbath',37),(38,1,'Have you been diagnosed for a skin condition in the past?',38),(39,1,'Alopecia (hair loss)',39),(40,1,'Acne',40),(42,1,'Eczema',41),(43,1,'Psoriasis',42),(44,1,'Rosacea',43),(45,1,'Skin cancer',44),(47,1,'Other',45),(48,1,'List any medical condition that you currently have or have been treated for:',46),(50,1,'Type to add a condition',47),(51,1,'Arthritis',48),(53,1,'Artifical Heart Valve',49),(55,1,'Artifical Joint',50),(56,1,'Asthma',51),(57,1,'Blood Clots',52),(58,1,'Diabetes',53),(59,1,'Epilepsy or Seizures',54),(60,1,'High blood pressure',55),(61,1,'High Cholestrol',56),(62,1,'HIV/AIDs',57),(63,1,'Heart Attack',58),(64,1,'Heart Murmur',59),(66,1,'Irregular Heartbeat',60),(67,1,'Kidney Disease',61),(68,1,'Liver disease',62),(69,1,'Lung Disease',63),(70,1,'Lupus',64),(71,1,'Organ Transplant',65),(72,1,'Pacemaker',66),(73,1,'Thyroid Problems',67),(74,1,'Other Condition Not Listed',68),(75,1,'No past or present conditions',69),(76,1,'Photos',87),(77,1,'Where do your breakouts occur?',70),(78,1,'Face',71),(79,1,'Chest',72),(80,1,'Back',73),(81,1,'Other',74),(82,1,'Up First: Face Photos',75),(83,1,'Remember these photos are for diagnosis purposes. The clearer your photo the easier it is for the doctor to make a diagnosis.',76),(84,1,'Remove glasses or hats',77),(85,1,'Pull back any hair covering your face',78),(86,1,'No make up',79),(87,1,'Remve any jewellery or clothing that may be covering your chest (except under garments)',81),(88,1,'Next: Chest Photos',80),(89,1,'Reason for visit',88),(90,1,'Length of time with acne symptoms',89),(91,1,'Other symptoms or conditions patient wants diagnosed',90),(92,1,'Worsening symptoms',91),(93,1,'Recent changes making acne worse',92),(94,1,'Type of treatments',93),(95,1,'Prescription tried',94),(96,1,'Additional info patient shared',95),(97,1,'Location of symptoms',96),(98,1,'Pregnancy',97),(99,1,'Medication allergies',98),(100,1,'Current medications',99),(101,1,'Social History',100),(102,1,'Skin conditions',101),(103,1,'Other Conditions',102),(104,1,'Or take a photo of the treatment',103),(105,1,'Face photos of patient',105),(106,1,'Chest photos of patient',106),(107,1,'Back photos of patient',107),(108,1,'Other photos of patient',108),(109,1,'Other',109),(110,1,'Face Front',82),(111,1,'Profile Left',83),(112,1,'Profile Right',84),(113,1,'Chest',85),(114,1,'How effective was this medication?',110),(115,1,'Not Very',111),(116,1,'Somewhat',112),(117,1,'Very',113),(118,1,'Are you currently using this medication?',114),(119,1,'0-1',115),(120,1,'2-5',116),(121,1,'6-11',117),(122,1,'12+',118),(123,1,'Not very effective',119),(124,1,'Somewhat',120),(125,1,'Very',121),(126,1,'Currently using it',122),(127,1,'Not currently using it',123),(128,1,'Used for less than 1 month',124),(129,1,'Used for 2-5 months',125),(131,1,'Used for 6-11 months',126),(132,1,'Used for over a year',127),(133,1,'Approximately how many months did you use this medication for?',128),(154,1,'When did you start getting acne breakouts?',150),(155,1,'During puberty',151),(156,1,'0-6 months ago',152),(157,1,'1-2 years ago',153),(158,1,'2 or more years ago',154),(159,1,'Onset of symptoms',155),(160,1,'Has your acne...',156),(161,1,'Been painful to the touch',157),(162,1,'Scarring',158),(163,1,'Caused discoloration',159),(164,1,'Types of symptoms',160),(165,1,'Does getting your period make your acne worse?',161),(166,1,'Are your periods regular?',162),(167,1,'Menstrual cycle',163),(168,1,'How would you describe your skin?',164),(169,1,'Normal',165),(170,1,'Oily',166),(171,1,'Dry',167),(172,1,'Combination',168),(173,1,'Skin type',169),(174,1,'Have you ever had an allergic reaction to a topical medication?',170),(175,1,'Topical Medication Allergies',171),(176,1,'Select which, if any, of the following conditions you have been treated for.',172),(177,1,'Other conditions',173),(178,1,'Gastritis',174),(179,1,'Colitis',175),(180,1,'Kidney disease',176),(181,1,'Lupus',177),(182,1,'Medication Allergies',104),(183,1,'Not very effective',178),(184,1,'Somewhat effective',179),(185,1,'Very effective',180),(186,1,'Not currently using it',181),(187,1,'Currently using it',182),(188,1,'Used for less than one month',183),(189,1,'Used for 2-5 months',184),(190,1,'Used for 6-11 months',185),(191,1,'Used for 12+ months',186),(192,1,'Add Medication',187),(193,1,'Add Medication',188),(194,1,'Or take a photo of the medication',189),(195,1,'Add Medication',190),(196,1,'Add Medication',191),(197,1,'Save Changes',192),(198,1,'Remove Medication',193),(199,1,'Remove Medication',194),(200,1,'What\'s your diagnosis?',195),(201,1,'Acne vulgaris',196),(202,1,'Acne rosacea',197),(203,1,'How severe is the patient\'s acne?',198),(204,1,'Mild',199),(205,1,'Moderate',200),(206,1,'Severe',201),(207,1,'What type of acne do they have?',202),(208,1,'Whiteheads',203),(209,1,'Pustules',204),(210,1,'Nodules',205),(211,1,'Inflammatory',206),(212,1,'Blackheads',207),(213,1,'Papules',208),(214,1,'Cystic',209),(215,1,'Hormonal',210),(216,1,'(select all that apply)',211),(217,1,'Bag',212),(218,1,'Bottle',213),(219,1,'Box',214),(220,1,'Capsule',215),(221,1,'Cartridge',216),(222,1,'Container',217),(223,1,'Drop',218),(224,1,'Gram',219),(225,1,'Inhaler',220),(226,1,'International',221),(227,1,'Kit',222),(228,1,'Liter',223),(229,1,'Lozenge',224),(230,1,'Milligram',225),(231,1,'Milliliter',226),(232,1,'Million Units',227),(233,1,'Mutually Defined',228),(234,1,'Fluid Ounce',229),(235,1,'Not Specified',230),(236,1,'Pack',231),(237,1,'Packet',232),(238,1,'Pint',233),(239,1,'Suppository',234),(240,1,'Syringe',235),(241,1,'Tablespoon',236),(242,1,'Tablet',237),(243,1,'Teaspoon',238),(244,1,'Transdermal Patch',239),(245,1,'Tube',240),(246,1,'Unit',241),(247,1,'Vial',242),(248,1,'Each',243),(249,1,'Gum',244),(250,1,'Ampule',245),(251,1,'Applicator',246),(252,1,'Applicatorful',247),(253,1,'Bar',248),(254,1,'Bead',249),(255,1,'Blister',250),(256,1,'Block',251),(257,1,'Bolus',252),(258,1,'Can',253),(259,1,'Canister',254),(260,1,'Capler',255),(261,1,'Carton',256),(262,1,'Case',257),(263,1,'Cassette',258),(264,1,'Cylinder',259),(265,1,'Disk',260),(266,1,'Dose Pack',261),(267,1,'Dual Packs',262),(268,1,'Film',263),(269,1,'Gallon',264),(270,1,'Implant',265),(271,1,'Inhalation',266),(272,1,'Inhaler Refill',267),(273,1,'Insert',268),(274,1,'Intravenous Bag',269),(275,1,'Milimeter',270),(276,1,'Nebule',271),(277,1,'Needle Free Injection',272),(278,1,'Oscular System',273),(279,1,'Ounce',274),(280,1,'Pad',275),(281,1,'Paper',276),(282,1,'Pouch',277),(283,1,'Pound',278),(284,1,'Puff',279),(285,1,'Quart',280),(286,1,'Ring',281),(287,1,'Sachet',282),(288,1,'Scoopful',283),(289,1,'Sponge',284),(290,1,'Spray',285),(291,1,'Stick',286),(292,1,'Strip',287),(293,1,'Swab',288),(294,1,'Tabminder',289),(295,1,'Tampon',290),(296,1,'Tray',291),(297,1,'Troche',292),(298,1,'Wafer',293),(299,1,'We\'re currently only diagnosing and treating acne but will be adding support for more conditions soon.',294),(300,1,'Help infom what we add next by telling us what your visit today was for...',295),(301,1,'Submit',296),(302,1,'Turned into cysts',297),(303,1,'None of the above',298),(304,1,'Did this medication irritate your skin?',299),(305,1,'Irritated skin',300),(306,1,'Did not irritate skin',301),(307,1,'Pregnant',302),(308,1,'Nursing',303),(309,1,'Planning a pregnancy',304),(310,1,'None of the above',305),(311,1,'How many months have you been taking this medication?',306),(312,1,'Taken for less than 1 month',307),(313,1,'Taken for 2-5 months',308),(314,1,'Taken for 6-11 months',309),(315,1,'Taken for 12+ months',310),(316,1,'Hypertension',311),(317,1,'Polycystic ovary syndrome',312),(318,1,'What skin condition(s) were you diagnosed with?',313),(319,1,'Neck',314),(320,1,'Acne mainly occurs on the face, neck, chest and back.\n\nIf the doctor determines that you have a condition other than acne you may be asked to visit a local dermatologist\'s office.',315),(321,1,'Type to add a location...',316),(322,1,'Type to add another condition',317),(323,1,'Regular periods',318),(324,1,'Worse with period',319),(325,1,'Recent changes',320),(326,1,'Currently pregnant',321),(327,1,'Currently nursing',322),(328,1,'Currently planning a pregnancy',323),(329,1,'Not currently pregnant, planning a pregnancy or nursing',324),(330,1,'Other skin condition specified',325),(331,1,'Perioral dermatitis',326),(332,1,'Comedonal',327),(333,1,'Erythematotelangiectatic',328),(334,1,'Papulopustular',329),(335,1,'Rhinophyma',330),(336,1,'Ocular',331),(337,1,'acne',332),(338,1,'6-12 months ago',333),(339,1,'Are you currently taking any medications?',334),(340,1,'List any other than those you may be using for acne.',335),(341,1,'None',336),(342,1,'Other location specified',337),(343,1,'No medications specified',338),(344,1,'No medications specified',339),(345,1,'None',340),(346,1,'Patient chose not to answer',341),(347,1,'No prescriptions tried',342),(348,1,'Patient chose not to answer',343),(349,1,'Currently XXX',344),(350,1,'Allergic to XXX',345),(351,1,'Which medications are you allergic to?',346),(352,1,'What\'s your preference for prescription medications?',347),(353,1,'Generic only',348),(354,1,'No preference',349),(355,1,'Generic Rxs only',350),(356,1,'Prescription Preference',351),(357,1,'Intestinal inflammation',352),(358,1,'Organ transplant',353),(359,1,'Many acne medications shouldn\'t be taken while pregnant or nursing.',354),(360,1,'No, I\'m not and will notify my doctor if I become pregnant during treatment',355),(361,1,'Been picked or squeezed',356),(362,1,'Created scars',357),(363,1,'Has a doctor ever prescribed medication to treat your acne?',358),(364,1,'Have you tried over-the-counter acne treatments?',359),(365,1,'List the products that you are current using or have tried in the past.',360),(366,1,'This question is optional but is your chance to let the doctor know what\'s on your mind.',361),(367,1,'Are you currently using this product?',362),(368,1,'How effective was this product?',363),(369,1,'Did this product irritate your skin?',364),(370,1,'Approximately how many months did you use this product for?',365),(371,1,'Add Product',366),(372,1,'Remove Product',367),(373,1,'Type to add a product',368),(374,1,'OTC products tried',371),(375,1,'Pregnant, planning a pregnancy, or nursing',372),(376,1,'Not Pregnant',373),(377,1,'No products tried',369),(378,1,'Not Suitable For Spruce',374),(379,1,'Describe the patient\'s condition:',375),(380,1,'Type your diagnosis',376),(381,1,'Why isn\'t this visit suitable for Spruce?',377),(382,1,'For internal purposes only, not shared with patient',378),(383,1,'Describe why you\'re not able to treat this case',379),(384,1,'Other Location',380),(385,1,'Select which acne medications you were prescribed.',381),(386,1,'BenzaClin',382),(387,1,'Benzoyl Peroxide',383),(388,1,'Clindamycin',384),(389,1,'Differin',385),(390,1,'Duac',386),(391,1,'Epiduo',387),(392,1,'Metrogel',388),(393,1,'Minocycline',389),(394,1,'Retin-A or Tretinoin',390),(395,1,'Tetracycline',391),(396,1,'Are you currently using it?',392),(397,1,'How effective was it?',393),(398,1,'Not very',394),(399,1,'Not Effective',395),(400,1,'Did you use it for more than three months?',396),(401,1,'Used for more than 3 months',397),(402,1,'Not used for more than 3 months',398),(403,1,'Did it cause irritation or an adverse reaction?',399),(404,1,'Anything else you\'d like to tell the doctor about it?',400),(405,1,'Optional...',401),(406,1,'Select which over-the-counter acne treatments you have tried.',402),(407,1,'Acne Free',403),(408,1,'Cetaphil',404),(409,1,'Clean and Clear',405),(410,1,'Clearasil',406),(411,1,'Noxzema',407),(412,1,'Oxy',408),(413,1,'Proactiv',409),(414,1,'Zeno',410),(415,1,'Type another treatment name',411),(416,1,'What <parent_answer_text> product have you tried?',412),(417,1,'Currently Using',413),(418,1,'Effective',414),(419,1,'Used for 3+ months',415),(420,1,'Irritating',416),(421,1,'Comments',417),(422,1,'Which product',418),(423,1,'How long',419),(424,1,'Face Side',420),(425,1,'Aveeno',421),(426,1,'PanOxyl',422),(427,1,'Doxycycline',423),(428,1,'How does your skin compare to the photos you took?',424),(429,1,'Usual skin compared to photos',431),(430,1,'I usually have more acne blemishes.',425),(431,1,'Has more acne blemishes',426),(432,1,'I usually have fewer acne blemishes.',427),(433,1,'Has fewer acne blemishes',428),(434,1,'My skin usually looks about the same.',429),(435,1,'Looks about the same',430),(436,1,'What type of medications does your insurance cover?',432),(437,1,'Insurance coverage',437),(438,1,'Brand name and generic',433),(439,1,'Generic only',434),(440,1,'I don\'t know',435),(441,1,'I don\'t have insurance',436),(442,1,'No insurance',438),(443,1,'Patient doesn\'t know',439),(444,1,'Sensitive',440),(445,1,'Type another description',441),(446,1,'Formed deep, hard lumps',442),(447,1,'Do you think your acne is made worse by any of the following?',443),(448,1,'Perceived contributing factors',444),(449,1,'Diet',445),(450,1,'Hair products',446),(451,1,'Makeup',447),(452,1,'Hormonal changes',448),(453,1,'Stress',449),(454,1,'Sweating or sports',450),(455,1,'Weather',451),(456,1,'I\'m not sure',452),(457,1,'Unsure',453),(458,1,'Type another factor',454),(459,1,'Neutrogena',455),(460,1,'Type another condition',456),(461,1,'How happy are you with the improvements in your skin?',457),(462,1,'Satisfaction level with improvements',458),(463,1,'Very happy',459),(464,1,'Happy',460),(465,1,'Neutral',461),(466,1,'Unhappy',462),(467,1,'Very Unhappy',463),(468,1,'Why aren\'t you happy with the improvements in your skin?',464),(469,1,'Comments',465),(470,1,'Patient chose not to answer',466),(471,1,'This will help your doctor make any necessary adjustments to your plan.',467),(472,1,'Overall have you been following your treatment plan as instructed?',468),(473,1,'Compliance with Treatment Plan',469),(474,1,'Yes, completely',470),(475,1,'Mostly',471),(476,1,'I\'m not sure',472),(477,1,'Compliant',473),(478,1,'Mostly compliant',474),(479,1,'Somewhat compliant',475),(480,1,'Not compliant',476),(481,1,'Not sure',477),(482,1,'Have you experienced any side effects from medications in your treatment plan?',478),(483,1,'Side effects from medications',479),(484,1,'Describe the side effects you experienced and which medications caused them.',480),(485,1,'Description.',481),(486,1,'Are you currently using all of the treatments prescribed in your plan?',482),(487,1,'Using all treatments prescribed in plan',483),(488,1,'Which treatments have you stopped using and why?',484),(489,1,'Comments',485),(490,1,'Has any part of your treatment plan been difficult to follow consistently?',486),(491,1,'Difficulty complying with treatment plan',487),(492,1,'Since beginning your treatment plan have you started taking any medications other than the ones prescribed for acne?',488),(493,1,'Add the other medications you are currently taking.',489),(494,1,'Current Medications',490),(495,1,'No medications specified',491),(496,1,'Optional, but let your doctor know if have any questions about how to use your treatment plan more effectively.',492),(497,1,'Since your last visit have you developed any medication allergies? ',493),(498,1,'Are there any changes to your medical history you think may be relevant for your doctor?',494),(499,1,'Other changes to medical history',495),(500,1,'Treatment Plan',496),(501,1,'Please describe the changes to your medical history:',497),(502,1,'Comments',498),(503,1,'Let your doctor know if have any questions about how to use your treatment plan more effectively.',499),(504,1,'sometimes',500),(505,1,'Severity',501),(506,1,'Mild',502),(507,1,'Moderate',503),(508,1,'Severe',504),(509,1,'Type',505),(510,1,'Cystic',506),(511,1,'Hormonal',507),(512,1,'How did we do?',508),(513,1,'Please rate your Spruce experience so far:',509),(514,1,'Anything else you\'d like to tell us?',510),(515,1,'Done',511);
/*!40000 ALTER TABLE `localized_text` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `answer_type`
--

DROP TABLE IF EXISTS `answer_type`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `answer_type` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `atype` varchar(250) NOT NULL,
  `deprecated` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `otype` (`atype`)
) ENGINE=InnoDB AUTO_INCREMENT=18 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `answer_type`
--

LOCK TABLES `answer_type` WRITE;
/*!40000 ALTER TABLE `answer_type` DISABLE KEYS */;
INSERT INTO `answer_type` VALUES (1,'a_type_multiple_choice',0),(2,'a_type_free_text',1),(3,'a_type_single_entry',1),(4,'a_type_dropdown_entry',1),(5,'a_type_autocomplete_entry',1),(6,'a_type_photo_to_text_entry',1),(7,'a_type_photo_entry_face_middle',1),(8,'a_type_photo_entry_face_left',1),(10,'a_type_photo_entry_face_right',1),(11,'a_type_photo_entry_back',1),(12,'a_type_photo_entry_chest',1),(13,'a_type_photo_entry_other',1),(14,'a_type_segmented_control',0),(15,'a_type_multiple_choice_none',0),(16,'a_type_photo_entry_neck',1),(17,'a_type_multiple_choice_other_free_text',0);
/*!40000 ALTER TABLE `answer_type` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `region`
--

DROP TABLE IF EXISTS `region`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `region` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `region_tag` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `region_tag` (`region_tag`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `region`
--

LOCK TABLES `region` WRITE;
/*!40000 ALTER TABLE `region` DISABLE KEYS */;
INSERT INTO `region` VALUES (1,'us-east-1'),(2,'us-west-1');
/*!40000 ALTER TABLE `region` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `languages_supported`
--

DROP TABLE IF EXISTS `languages_supported`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `languages_supported` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `language` varchar(10) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `languages_supported`
--

LOCK TABLES `languages_supported` WRITE;
/*!40000 ALTER TABLE `languages_supported` DISABLE KEYS */;
INSERT INTO `languages_supported` VALUES (1,'en');
/*!40000 ALTER TABLE `languages_supported` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `tips`
--

DROP TABLE IF EXISTS `tips`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `tips` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `tips_text_id` int(10) unsigned NOT NULL,
  `tips_tag` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `tips_tag` (`tips_tag`),
  KEY `tips_text_id` (`tips_text_id`),
  CONSTRAINT `tips_ibfk_1` FOREIGN KEY (`tips_text_id`) REFERENCES `app_text` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `tips`
--

LOCK TABLES `tips` WRITE;
/*!40000 ALTER TABLE `tips` DISABLE KEYS */;
INSERT INTO `tips` VALUES (1,77,'tip_remove_glasses'),(2,78,'tip_pull_hair_back'),(3,79,'tip_no_make_up'),(4,81,'tip_remove_jewellery');
/*!40000 ALTER TABLE `tips` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `tips_section`
--

DROP TABLE IF EXISTS `tips_section`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `tips_section` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `tips_section_tag` varchar(100) NOT NULL,
  `comment` varchar(500) DEFAULT NULL,
  `tips_title_text_id` int(10) unsigned NOT NULL,
  `tips_subtext_text_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `tips_section_tag` (`tips_section_tag`),
  KEY `tips_title_text_id` (`tips_title_text_id`),
  KEY `tips_subtext_text_id` (`tips_subtext_text_id`),
  CONSTRAINT `tips_section_ibfk_1` FOREIGN KEY (`tips_subtext_text_id`) REFERENCES `app_text` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `tips_section`
--

LOCK TABLES `tips_section` WRITE;
/*!40000 ALTER TABLE `tips_section` DISABLE KEYS */;
INSERT INTO `tips_section` VALUES (1,'tips_section_face','tips for taking pictures of face',75,76),(2,'tips_section_chest','tips for taking pictures of chest',80,76);
/*!40000 ALTER TABLE `tips_section` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `section`
--

DROP TABLE IF EXISTS `section`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `section` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `section_title_app_text_id` int(10) unsigned NOT NULL,
  `comment` varchar(600) NOT NULL,
  `section_tag` varchar(250) NOT NULL,
  `clinical_pathway_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `section_tag` (`section_tag`),
  KEY `section_title_app_text_id` (`section_title_app_text_id`),
  KEY `clinical_pathway_id` (`clinical_pathway_id`),
  CONSTRAINT `section_ibfk_1` FOREIGN KEY (`section_title_app_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `section_ibfk_3` FOREIGN KEY (`clinical_pathway_id`) REFERENCES `clinical_pathway` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `section`
--

LOCK TABLES `section` WRITE;
/*!40000 ALTER TABLE `section` DISABLE KEYS */;
INSERT INTO `section` VALUES (1,30,'skin history section','section_skin_history',1),(2,31,'medical history section','section_medical_history',1),(3,87,'photos for diagnosis','section_photo_diagnosis',1),(4,496,'treatment plan section','section_treatment_plan',1),(5,31,'followup medical history section','section_followup_medical_history',1);
/*!40000 ALTER TABLE `section` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `screen_type`
--

DROP TABLE IF EXISTS `screen_type`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `screen_type` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `screen_type_tag` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `screen_type_tag` (`screen_type_tag`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `screen_type`
--

LOCK TABLES `screen_type` WRITE;
/*!40000 ALTER TABLE `screen_type` DISABLE KEYS */;
INSERT INTO `screen_type` VALUES (1,'screen_type_general'),(2,'screen_type_photo');
/*!40000 ALTER TABLE `screen_type` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `question_type`
--

DROP TABLE IF EXISTS `question_type`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `question_type` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `qtype` varchar(250) DEFAULT NULL,
  `deprecated` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `qtype` (`qtype`)
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `question_type`
--

LOCK TABLES `question_type` WRITE;
/*!40000 ALTER TABLE `question_type` DISABLE KEYS */;
INSERT INTO `question_type` VALUES (1,'q_type_multiple_choice',0),(2,'q_type_free_text',0),(3,'q_type_compound',1),(4,'q_type_single_photo',1),(5,'q_type_single_entry',1),(6,'q_type_single_select',0),(7,'q_type_multiple_photo',1),(8,'q_type_segmented_control',0),(9,'q_type_autocomplete',0),(10,'q_type_photo',1),(11,'q_type_photo_section',0);
/*!40000 ALTER TABLE `question_type` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `question`
--

DROP TABLE IF EXISTS `question`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `question` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `qtext_app_text_id` int(10) unsigned DEFAULT NULL,
  `qtext_short_text_id` int(10) unsigned DEFAULT NULL,
  `subtext_app_text_id` int(10) unsigned DEFAULT NULL,
  `question_tag` varchar(250) NOT NULL,
  `parent_question_id` int(10) unsigned DEFAULT NULL,
  `required` tinyint(1) DEFAULT NULL,
  `formatted_field_tags` varchar(150) DEFAULT NULL,
  `to_alert` tinyint(1) DEFAULT NULL,
  `alert_app_text_id` int(10) unsigned DEFAULT NULL,
  `qtext_has_tokens` tinyint(1) DEFAULT NULL,
  `language_id` int(10) unsigned DEFAULT '1',
  `version` int(10) unsigned NOT NULL DEFAULT '1',
  `summary_text` varchar(600) DEFAULT NULL,
  `subtext_text` varchar(600) DEFAULT NULL,
  `question_text` varchar(600) DEFAULT NULL,
  `alert_text` varchar(600) DEFAULT NULL,
  `question_type` varchar(60) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_question_question_tag_version` (`question_tag`,`version`,`language_id`),
  KEY `subtext_app_text_id` (`subtext_app_text_id`),
  KEY `qtext_app_text_id` (`qtext_app_text_id`),
  KEY `qtext_short_text_id` (`qtext_short_text_id`),
  KEY `parent_question_id` (`parent_question_id`),
  KEY `alert_app_text_id` (`alert_app_text_id`),
  KEY `fk_question_languages_supported_id` (`language_id`),
  CONSTRAINT `fk_question_languages_supported_id` FOREIGN KEY (`language_id`) REFERENCES `languages_supported` (`id`),
  CONSTRAINT `question_ibfk_2` FOREIGN KEY (`subtext_app_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `question_ibfk_3` FOREIGN KEY (`qtext_app_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `question_ibfk_4` FOREIGN KEY (`qtext_short_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `question_ibfk_5` FOREIGN KEY (`parent_question_id`) REFERENCES `question` (`id`),
  CONSTRAINT `question_ibfk_6` FOREIGN KEY (`alert_app_text_id`) REFERENCES `app_text` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=95 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `question`
--

LOCK TABLES `question` WRITE;
/*!40000 ALTER TABLE `question` DISABLE KEYS */;
INSERT INTO `question` VALUES (1,1,88,NULL,'q_reason_visit',NULL,NULL,'title:doctor_last_name',NULL,NULL,NULL,1,1,'Reason for visit',NULL,'What\'s the reason for your visit with Dr. %s today?',NULL,'q_type_single_select'),(2,294,90,NULL,'q_condition_for_diagnosis',NULL,NULL,'',NULL,NULL,NULL,1,1,'Other symptoms or conditions patient wants diagnosed',NULL,'We\'re currently only diagnosing and treating acne but will be adding support for more conditions soon.',NULL,'q_type_single_entry'),(3,5,89,NULL,'q_acne_length',NULL,NULL,'',NULL,NULL,NULL,1,1,'Length of time with acne symptoms',NULL,'How long have you been experiencing acne symptoms?',NULL,'q_type_single_select'),(4,10,91,NULL,'q_acne_worse',NULL,1,'',NULL,NULL,NULL,1,1,'Worsening symptoms',NULL,'Has your acne been getting worse?',NULL,'q_type_single_select'),(6,13,320,NULL,'q_changes_acne_worse',NULL,0,'',NULL,NULL,NULL,1,1,'Recent changes',NULL,'Are there any recent changes that could be affecting your skin?',NULL,'q_type_free_text'),(7,15,93,NULL,'q_acne_prev_treatment_types',NULL,1,'',NULL,NULL,NULL,1,1,'Type of treatments',NULL,'What types of treatments have you previously tried for your acne?',NULL,'q_type_multiple_choice'),(8,19,94,NULL,'q_acne_prev_treatment_list',8,1,'',NULL,NULL,NULL,1,1,'Prescription tried',NULL,'List medications that you are currently using or have tried in the past.',NULL,'q_type_autocomplete'),(9,21,95,NULL,'q_anything_else_acne',NULL,0,NULL,NULL,NULL,NULL,1,1,'Additional info patient shared',NULL,'Is there anything else you\'d like to share about your skin?',NULL,'q_type_free_text'),(10,23,97,354,'q_pregnancy_planning',NULL,1,'',1,344,NULL,1,1,'Pregnancy','Many acne medications shouldn\'t be taken while pregnant or nursing.','Are you pregnant, planning a pregnancy or nursing?','Currently XXX','q_type_single_select'),(11,26,98,NULL,'q_allergic_medications',NULL,1,'',NULL,NULL,NULL,1,1,'Medication allergies',NULL,'Are you allergic to any medications?',NULL,'q_type_single_select'),(12,346,98,NULL,'q_allergic_medication_entry',NULL,1,'',1,345,NULL,1,1,'Medication allergies',NULL,'Which medications are you allergic to?','Allergic to XXX','q_type_autocomplete'),(13,32,99,NULL,'q_current_medications_entry',13,1,'',NULL,NULL,NULL,1,1,'Current medications',NULL,'Which medications are you currently taking?',NULL,'q_type_autocomplete'),(14,34,100,NULL,'q_social_history',NULL,NULL,'',NULL,NULL,NULL,1,1,'Social History',NULL,'Select which if any of the following activities you do regularly:',NULL,'q_type_multiple_choice'),(15,38,101,NULL,'q_prev_skin_condition_diagnosis',NULL,1,'',NULL,NULL,NULL,1,1,'Skin conditions',NULL,'Have you been diagnosed for a skin condition in the past?',NULL,'q_type_single_select'),(16,46,102,NULL,'q_prev_med_condition_diagnosis',NULL,NULL,'',NULL,NULL,NULL,1,1,'Other Conditions',NULL,'List any medical condition that you currently have or have been treated for:',NULL,'q_type_compound'),(17,313,101,NULL,'q_list_prev_skin_condition_diagnosis',NULL,1,'',NULL,NULL,NULL,1,1,'Skin conditions',NULL,'What skin condition(s) were you diagnosed with?',NULL,'q_type_multiple_choice'),(18,70,96,NULL,'q_acne_location',NULL,1,'',NULL,NULL,NULL,1,1,'Location of symptoms',NULL,'Where do your breakouts occur?',NULL,'q_type_multiple_choice'),(19,NULL,105,NULL,'q_face_photo_intake',NULL,1,'',NULL,NULL,NULL,1,1,'Face photos of patient',NULL,NULL,NULL,'q_type_photo'),(20,NULL,106,NULL,'q_chest_photo_intake',NULL,1,'',NULL,NULL,NULL,1,1,'Chest photos of patient',NULL,NULL,NULL,'q_type_photo'),(21,NULL,107,NULL,'q_back_photo_intake',NULL,1,'',NULL,NULL,NULL,1,1,'Back photos of patient',NULL,NULL,NULL,'q_type_photo'),(22,NULL,108,NULL,'q_other_photo_intake',NULL,1,'',NULL,NULL,NULL,1,1,'Other photos of patient',NULL,NULL,NULL,'q_type_photo'),(24,110,NULL,NULL,'q_effective_treatment',8,1,'',NULL,NULL,NULL,1,1,NULL,NULL,'How effective was this medication?',NULL,'q_type_segmented_control'),(25,114,NULL,NULL,'q_using_treatment',8,1,'',NULL,NULL,NULL,1,1,NULL,NULL,'Are you currently using this medication?',NULL,'q_type_segmented_control'),(26,128,NULL,NULL,'q_length_treatment',8,1,'',NULL,NULL,NULL,1,1,NULL,NULL,'Approximately how many months did you use this medication for?',NULL,'q_type_segmented_control'),(28,150,155,NULL,'q_onset_acne',NULL,1,'',NULL,NULL,NULL,1,1,'Onset of symptoms',NULL,'When did you start getting acne breakouts?',NULL,'q_type_single_select'),(29,156,160,NULL,'q_acne_symptoms',NULL,1,'',NULL,NULL,NULL,1,1,'Types of symptoms',NULL,'Has your acne...',NULL,'q_type_multiple_choice'),(30,161,319,NULL,'q_acne_worse_period',NULL,1,'',NULL,NULL,NULL,1,1,'Worse with period',NULL,'Does getting your period make your acne worse?',NULL,'q_type_single_select'),(31,162,318,NULL,'q_periods_regular',NULL,1,'',NULL,NULL,NULL,1,1,'Regular periods',NULL,'Are your periods regular?',NULL,'q_type_single_select'),(32,164,169,NULL,'q_skin_description',NULL,1,'',NULL,NULL,NULL,1,1,'Skin type',NULL,'How would you describe your skin?',NULL,'q_type_multiple_choice'),(33,170,171,NULL,'q_topical_allergic_medications',NULL,1,'',NULL,NULL,NULL,1,1,'Topical Medication Allergies',NULL,'Have you ever had an allergic reaction to a topical medication?',NULL,'q_type_single_select'),(34,172,173,NULL,'q_other_conditions_acne',NULL,1,'',NULL,NULL,NULL,1,1,'Other conditions',NULL,'Select which, if any, of the following conditions you have been treated for.',NULL,'q_type_multiple_choice'),(36,29,171,NULL,'q_topical_allergies_medication_entry',NULL,0,'',NULL,NULL,NULL,1,1,'Topical Medication Allergies',NULL,'Type to add a medication',NULL,'q_type_autocomplete'),(37,195,NULL,NULL,'q_acne_diagnosis',NULL,1,'',NULL,NULL,NULL,1,1,NULL,NULL,'What\'s your diagnosis?',NULL,'q_type_single_select'),(38,198,NULL,NULL,'q_acne_severity',NULL,1,'',NULL,NULL,NULL,1,1,NULL,NULL,'How severe is the patient\'s acne?',NULL,'q_type_single_select'),(39,202,NULL,NULL,'q_acne_type',NULL,1,'',NULL,NULL,NULL,1,1,NULL,NULL,'What type of acne do they have?',NULL,'q_type_multiple_choice'),(40,299,NULL,NULL,'q_treatment_irritate_skin',NULL,1,'',NULL,NULL,NULL,1,1,NULL,NULL,'Did this medication irritate your skin?',NULL,'q_type_segmented_control'),(41,306,419,NULL,'q_length_current_medication',NULL,1,'',NULL,NULL,NULL,1,1,'How long',NULL,'How many months have you been taking this medication?',NULL,'q_type_segmented_control'),(42,NULL,NULL,NULL,'q_neck_photo_intake',NULL,NULL,'',NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL,'q_type_photo'),(43,315,337,NULL,'q_other_acne_location_entry',NULL,1,'',NULL,NULL,NULL,1,1,'Other location specified',NULL,'Acne mainly occurs on the face, neck, chest and back.\n\nIf the doctor determines that you have a condition other than acne you may be asked to visit a local dermatologist\'s office.',NULL,'q_type_single_entry'),(44,NULL,325,NULL,'q_other_skin_condition_entry',NULL,NULL,'',NULL,NULL,NULL,1,1,'Other skin condition specified',NULL,NULL,NULL,'q_type_single_entry'),(45,202,NULL,211,'q_acne_rosacea_type',NULL,1,'',NULL,NULL,NULL,1,1,NULL,'(select all that apply)','What type of acne do they have?',NULL,'q_type_multiple_choice'),(46,334,NULL,NULL,'q_current_medications',NULL,1,'',NULL,NULL,NULL,1,1,NULL,NULL,'Are you currently taking any medications?',NULL,'q_type_single_select'),(47,NULL,NULL,NULL,'q_face_left_photo_intake',NULL,1,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL,'q_type_photo'),(48,NULL,NULL,NULL,'q_face_right_photo_intake',NULL,1,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL,'q_type_photo'),(49,347,351,NULL,'q_prescription_preference',NULL,1,NULL,1,350,NULL,1,1,'Prescription Preference',NULL,'What\'s your preference for prescription medications?','Generic Rxs only','q_type_single_select'),(50,358,NULL,NULL,'q_acne_prev_prescriptions',NULL,1,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Has a doctor ever prescribed medication to treat your acne?',NULL,'q_type_single_select'),(51,359,NULL,NULL,'q_acne_prev_otc_treatments',NULL,1,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Have you tried over-the-counter acne treatments?',NULL,'q_type_single_select'),(52,360,371,NULL,'q_acne_prev_otc_list',NULL,1,NULL,NULL,NULL,NULL,1,1,'OTC products tried',NULL,'List the products that you are current using or have tried in the past.',NULL,'q_type_autocomplete'),(53,362,NULL,NULL,'q_using_otc',52,1,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Are you currently using this product?',NULL,'q_type_segmented_control'),(54,363,NULL,NULL,'q_effective_otc',52,1,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'How effective was this product?',NULL,'q_type_segmented_control'),(55,364,NULL,NULL,'q_otc_irritate_skin',52,1,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Did this product irritate your skin?',NULL,'q_type_segmented_control'),(56,365,NULL,NULL,'q_length_otc',52,1,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Approximately how many months did you use this product for?',NULL,'q_type_segmented_control'),(57,375,NULL,NULL,'q_diagnosis_describe_condition',NULL,1,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Describe the patient\'s condition:',NULL,'q_type_free_text'),(58,377,NULL,NULL,'q_diagnosis_reason_not_suitable',NULL,1,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Why isn\'t this visit suitable for Spruce?',NULL,'q_type_free_text'),(59,71,NULL,NULL,'q_face_photo_section',NULL,0,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Face',NULL,'q_type_photo_section'),(60,72,NULL,NULL,'q_chest_photo_section',NULL,0,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Chest',NULL,'q_type_photo_section'),(61,73,NULL,NULL,'q_back_photo_section',NULL,0,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Back',NULL,'q_type_photo_section'),(62,380,NULL,NULL,'q_other_location_photo_section',NULL,0,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Other Location',NULL,'q_type_photo_section'),(63,381,94,NULL,'q_acne_prev_prescriptions_select',NULL,1,NULL,NULL,NULL,NULL,1,1,'Prescription tried',NULL,'Select which acne medications you were prescribed.',NULL,'q_type_multiple_choice'),(64,392,413,NULL,'q_using_prev_acne_prescription',63,1,NULL,NULL,NULL,NULL,1,1,'Currently Using',NULL,'Are you currently using it?',NULL,'q_type_segmented_control'),(65,393,414,NULL,'q_how_effective_prev_acne_prescription',63,1,NULL,NULL,NULL,NULL,1,1,'Effective',NULL,'How effective was it?',NULL,'q_type_segmented_control'),(66,396,415,NULL,'q_use_more_three_months_prev_acne_prescription',63,1,NULL,NULL,NULL,NULL,1,1,'Used for 3+ months',NULL,'Did you use it for more than three months?',NULL,'q_type_segmented_control'),(67,399,416,NULL,'q_irritate_skin_prev_acne_prescription',63,1,NULL,NULL,NULL,NULL,1,1,'Irritating',NULL,'Did it cause irritation or an adverse reaction?',NULL,'q_type_segmented_control'),(68,400,417,NULL,'q_anything_else_prev_acne_prescription',63,0,NULL,NULL,NULL,NULL,1,1,'Comments',NULL,'Anything else you\'d like to tell the doctor about it?',NULL,'q_type_free_text'),(69,402,371,NULL,'q_acne_prev_otc_select',NULL,1,NULL,NULL,NULL,NULL,1,1,'OTC products tried',NULL,'Select which over-the-counter acne treatments you have tried.',NULL,'q_type_multiple_choice'),(70,412,418,NULL,'q_acne_otc_product_tried',69,0,NULL,NULL,NULL,1,1,1,'Which product',NULL,'What <parent_answer_text> product have you tried?',NULL,'q_type_single_entry'),(71,392,413,NULL,'q_using_prev_acne_otc',69,1,NULL,NULL,NULL,NULL,1,1,'Currently Using',NULL,'Are you currently using it?',NULL,'q_type_segmented_control'),(72,393,414,NULL,'q_how_effective_prev_acne_otc',69,1,NULL,NULL,NULL,NULL,1,1,'Effective',NULL,'How effective was it?',NULL,'q_type_segmented_control'),(73,399,416,NULL,'q_irritate_skin_prev_acne_otc',69,1,NULL,NULL,NULL,NULL,1,1,'Irritating',NULL,'Did it cause irritation or an adverse reaction?',NULL,'q_type_segmented_control'),(74,400,417,NULL,'q_anything_else_prev_acne_otc',69,0,NULL,NULL,NULL,NULL,1,1,'Comments',NULL,'Anything else you\'d like to tell the doctor about it?',NULL,'q_type_free_text'),(75,424,431,NULL,'q_skin_photo_comparison',NULL,1,NULL,NULL,NULL,NULL,1,1,'Usual skin compared to photos',NULL,'How does your skin compare to the photos you took?',NULL,'q_type_single_select'),(76,432,437,NULL,'q_insurance_coverage',NULL,1,NULL,NULL,NULL,NULL,1,1,'Insurance coverage',NULL,'What type of medications does your insurance cover?',NULL,'q_type_single_select'),(77,443,444,NULL,'q_acne_worse_contributing_factors',NULL,0,NULL,NULL,NULL,NULL,1,1,'Perceived contributing factors',NULL,'Do you think your acne is made worse by any of the following?',NULL,'q_type_multiple_choice'),(78,457,458,NULL,'q_skin_improvements',NULL,1,NULL,NULL,NULL,NULL,1,1,'Satisfaction level with improvements',NULL,'How happy are you with the improvements in your skin?',NULL,'q_type_single_select'),(79,464,465,NULL,'q_skin_improvements_why_not_happy',NULL,1,NULL,NULL,NULL,NULL,1,1,'Comments',NULL,'Why aren\'t you happy with the improvements in your skin?',NULL,'q_type_free_text'),(80,468,469,NULL,'q_using_tp_as_instructed',NULL,1,NULL,NULL,NULL,NULL,1,1,'Compliance with Treatment Plan',NULL,'Overall have you been following your treatment plan as instructed?',NULL,'q_type_single_select'),(81,478,479,NULL,'q_side_effects_from_tp',NULL,1,NULL,NULL,NULL,NULL,1,1,'Side effects from medications',NULL,'Have you experienced any side effects from medications in your treatment plan?',NULL,'q_type_single_select'),(82,480,417,NULL,'q_side_effects_from_tp_explain',NULL,1,NULL,NULL,NULL,NULL,1,1,'Comments',NULL,'Describe the side effects you experienced and which medications caused them.',NULL,'q_type_free_text'),(83,482,483,NULL,'q_using_all_treatments_in_tp',NULL,1,NULL,NULL,NULL,NULL,1,1,'Using all treatments prescribed in plan',NULL,'Are you currently using all of the treatments prescribed in your plan?',NULL,'q_type_single_select'),(84,484,485,NULL,'q_treatments_in_tp_stopped_using',NULL,1,NULL,NULL,NULL,NULL,1,1,'Comments',NULL,'Which treatments have you stopped using and why?',NULL,'q_type_free_text'),(85,486,487,NULL,'q_tp_compliance_difficulty',NULL,0,NULL,NULL,NULL,NULL,1,1,'Difficulty complying with treatment plan',NULL,'Has any part of your treatment plan been difficult to follow consistently?',NULL,'q_type_free_text'),(86,488,NULL,NULL,'q_other_medications_since_tp',NULL,1,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Since beginning your treatment plan have you started taking any medications other than the ones prescribed for acne?',NULL,'q_type_single_select'),(87,489,490,NULL,'q_other_medications_since_tp_entry',NULL,1,NULL,NULL,NULL,NULL,1,1,'Current Medications',NULL,'Add the other medications you are currently taking.',NULL,'q_type_autocomplete'),(88,493,NULL,NULL,'q_medication_allergies_since_visit',NULL,1,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Since your last visit have you developed any medication allergies? ',NULL,'q_type_single_select'),(89,497,495,NULL,'q_med_hx_changes_relevant_description',NULL,1,NULL,NULL,NULL,NULL,1,1,'Other changes to medical history',NULL,'Please describe the changes to your medical history:',NULL,'q_type_free_text'),(90,494,495,NULL,'q_med_hx_changes_relevant',NULL,1,NULL,NULL,NULL,NULL,1,1,'Other changes to medical history',NULL,'Are there any changes to your medical history you think may be relevant for your doctor?',NULL,'q_type_single_select'),(91,346,98,NULL,'q_medication_allergies_since_visit_entry',NULL,1,NULL,1,345,NULL,1,1,'Medication allergies',NULL,'Which medications are you allergic to?','Allergic to XXX','q_type_autocomplete'),(92,501,NULL,NULL,'q_diagnosis_severity',NULL,0,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Severity',NULL,'q_type_single_select'),(93,505,NULL,NULL,'q_diagnosis_acne_vulgaris_type',NULL,0,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Type',NULL,'q_type_multiple_choice'),(94,505,NULL,NULL,'q_diagnosis_acne_rosacea_type',NULL,0,NULL,NULL,NULL,NULL,1,1,NULL,NULL,'Type',NULL,'q_type_multiple_choice');
/*!40000 ALTER TABLE `question` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `additional_question_fields`
--

DROP TABLE IF EXISTS `additional_question_fields`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `additional_question_fields` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `question_id` int(10) unsigned NOT NULL,
  `json` blob NOT NULL,
  `language_id` int(10) unsigned DEFAULT '1',
  PRIMARY KEY (`id`),
  KEY `fk_additional_answer_fields_question_id` (`question_id`),
  CONSTRAINT `fk_additional_answer_fields_question_id` FOREIGN KEY (`question_id`) REFERENCES `question` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=129 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `additional_question_fields`
--

LOCK TABLES `additional_question_fields` WRITE;
/*!40000 ALTER TABLE `additional_question_fields` DISABLE KEYS */;
INSERT INTO `additional_question_fields` VALUES (1,62,'{\"allows_multiple_sections\":true, \"user_defined_section_title\":true}',1),(2,12,'{\"add_text\":\"Used for less than one month\"}',1),(3,12,'{\"placeholder_text\":\"Type to add a medication\"}',1),(4,12,'{\"add_photo_text\":\"Used for 2-5 months\"}',1),(5,13,'{\"add_text\":\"Used for less than one month\"}',1),(6,13,'{\"placeholder_text\":\"Type to add a medication\"}',1),(7,13,'{\"add_photo_text\":\"Used for 2-5 months\"}',1),(8,36,'{\"add_text\":\"Used for less than one month\"}',1),(9,36,'{\"placeholder_text\":\"Type to add a medication\"}',1),(10,36,'{\"add_photo_text\":\"Used for 2-5 months\"}',1),(11,8,'{\"add_text\":\"Currently using it\"}',1),(12,8,'{\"placeholder_text\":\"Over the counter\"}',1),(13,8,'{\"add_photo_text\":\"Other Conditions\"}',1),(14,8,'{\"add_button_text\":\"Used for 12+ months\"}',1),(15,8,'{\"save_button_text\":\"Add Medication\"}',1),(16,8,'{\"remove_button_text\":\"Add Medication\"}',1),(17,13,'{\"add_button_text\":\"Used for 6-11 months\"}',1),(18,13,'{\"save_button_text\":\"Add Medication\"}',1),(19,13,'{\"remove_button_text\":\"Add Medication\"}',1),(20,12,'{\"add_button_text\":\"Used for 6-11 months\"}',1),(21,12,'{\"save_button_text\":\"Add Medication\"}',1),(22,12,'{\"remove_button_text\":\"Add Medication\"}',1),(23,36,'{\"add_button_text\":\"Used for 6-11 months\"}',1),(24,36,'{\"save_button_text\":\"Add Medication\"}',1),(25,36,'{\"remove_button_text\":\"Add Medication\"}',1),(26,6,'{\"placeholder_text\":\"Yes\"}',1),(27,9,'{\"placeholder_text\":\"No treatments tried\"}',1),(28,2,'{\"placeholder_text\":\"Tampon\"}',1),(29,2,'{\"submit_button_text\":\"Tray\"}',1),(30,43,'{\"placeholder_text\":\"Hypertension\"}',1),(31,44,'{\"placeholder_text\":\"Polycystic ovary syndrome\"}',1),(32,12,'{\"empty_state_text\":\"6-12 months ago\"}',1),(33,13,'{\"empty_state_text\":\"Are you currently taking any medications?\"}',1),(34,17,'{\"empty_state_text\":\"List any other than those you may be using for acne.\"}',1),(35,6,'{\"empty_state_text\":\"None\"}',1),(36,8,'{\"empty_state_text\":\"Other location specified\"}',1),(37,9,'{\"empty_state_text\":\"No medications specified\"}',1),(38,9,'{\"placeholder_text\":\"Been picked or squeezed\"}',1),(39,52,'{\"add_text\":\"This question is optional but is your chance to let the doctor know what\'s on your mind.\"}',1),(40,52,'{\"placeholder_text\":\"How effective was this product?\"}',1),(41,52,'{\"add_button_text\":\"This question is optional but is your chance to let the doctor know what\'s on your mind.\"}',1),(42,52,'{\"save_button_text\":\"Add Medication\"}',1),(43,52,'{\"remove_button_text\":\"Are you currently using this product?\"}',1),(44,52,'{\"empty_state_text\":\"Did this product irritate your skin?\"}',1),(45,57,'{\"placeholder_text\":\"Not Pregnant\"}',1),(46,58,'{\"placeholder_text\":\"Describe the patient\'s condition:\"}',1),(47,68,'{\"placeholder_text\":\"Used for more than 3 months\"}',1),(48,69,'{\"other_answer_placeholder_text\":\"Noxzema\"}',1),(49,69,'{\"other_answer_placeholder_text\":\"Noxzema\"}',1),(50,74,'{\"placeholder_text\":\"Used for more than 3 months\"}',1),(51,63,'{\"empty_state_text\":\"Other location specified\"}',1),(52,69,'{\"empty_state_text\":\"Did this product irritate your skin?\"}',1),(53,63,'{\"other_answer_placeholder_text\":\"Noxzema\"}',1),(54,70,'{\"placeholder_text\":\"Used for more than 3 months\"}',1),(55,32,'{\"other_answer_placeholder_text\":\"I don\'t have insurance\"}',1),(56,77,'{\"other_answer_placeholder_text\":\"Sweating or sports\"}',1),(57,17,'{\"other_answer_placeholder_text\":\"I\'m not sure\"}',1),(58,79,'{\"empty_state_text\":\"Unhappy\"}',1),(59,79,'{\"placeholder_text\":\"Very Unhappy\"}',1),(60,82,'{\"empty_state_text\":\"Unhappy\"}',1),(61,82,'{\"placeholder_text\":\"Very Unhappy\"}',1),(62,84,'{\"empty_state_text\":\"Unhappy\"}',1),(63,84,'{\"placeholder_text\":\"Very Unhappy\"}',1),(64,87,'{\"empty_state_text\":\"Difficulty complying with treatment plan\"}',1),(65,87,'{\"placeholder_text\":\"Since beginning your treatment plan have you started taking any medications other than the ones prescribed for acne?\"}',1),(66,87,'{\"add_button_text\":\"Used for less than one month\"}',1),(67,87,'{\"add_button_text\":\"Used for less than one month\"}',1),(68,87,'{\"placeholder_text\":\"Type to add a medication\"}',1),(69,89,'{\"empty_state_text\":\"Unhappy\"}',1),(70,89,'{\"placeholder_text\":\"Very Unhappy\"}',1),(71,87,'{\"add_text\":\"Used for less than one month\"}',1),(72,87,'{\"save_button_text\":\"Add Medication\"}',1),(73,87,'{\"remove_button_text\":\"Add Medication\"}',1),(74,85,'{\"placeholder_text\":\"Other changes to medical history\"}',1),(75,91,'{\"add_text\":\"Used for less than one month\"}',1),(76,91,'{\"placeholder_text\":\"Type to add a medication\"}',1),(77,91,'{\"add_photo_text\":\"Used for 2-5 months\"}',1),(78,91,'{\"add_button_text\":\"Used for 6-11 months\"}',1),(79,91,'{\"save_button_text\":\"Add Medication\"}',1),(80,91,'{\"remove_button_text\":\"Add Medication\"}',1),(81,91,'{\"empty_state_text\":\"6-12 months ago\"}',1);
/*!40000 ALTER TABLE `additional_question_fields` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `potential_answer`
--

DROP TABLE IF EXISTS `potential_answer`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `potential_answer` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `question_id` int(10) unsigned NOT NULL,
  `answer_localized_text_id` int(10) unsigned DEFAULT NULL,
  `potential_answer_tag` varchar(250) NOT NULL,
  `ordering` int(10) unsigned NOT NULL,
  `answer_summary_text_id` int(10) unsigned DEFAULT NULL,
  `status` varchar(100) NOT NULL,
  `to_alert` tinyint(1) DEFAULT NULL,
  `language_id` int(10) unsigned NOT NULL DEFAULT '1',
  `answer_text` varchar(600) DEFAULT NULL,
  `answer_summary_text` varchar(600) DEFAULT NULL,
  `answer_type` varchar(60) NOT NULL,
  `client_data` blob,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_potential_answer_tag_quid_order` (`potential_answer_tag`,`question_id`,`ordering`,`language_id`),
  KEY `outcome_localized_text` (`answer_localized_text_id`),
  KEY `answer_summary_text_id` (`answer_summary_text_id`),
  KEY `fk_potential_answer_languages_supported_id` (`language_id`),
  KEY `fk_question_question_id` (`question_id`),
  CONSTRAINT `fk_potential_answer_languages_supported_id` FOREIGN KEY (`language_id`) REFERENCES `languages_supported` (`id`),
  CONSTRAINT `fk_question_question_id` FOREIGN KEY (`question_id`) REFERENCES `question` (`id`),
  CONSTRAINT `potential_answer_ibfk_3` FOREIGN KEY (`answer_summary_text_id`) REFERENCES `app_text` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=258 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `potential_answer`
--

LOCK TABLES `potential_answer` WRITE;
/*!40000 ALTER TABLE `potential_answer` DISABLE KEYS */;
INSERT INTO `potential_answer` VALUES (1,1,2,'a_acne',0,NULL,'ACTIVE',NULL,1,'Acne',NULL,'a_type_multiple_choice',NULL),(2,1,3,'a_something_else',1,NULL,'ACTIVE',NULL,1,'Other',NULL,'a_type_multiple_choice',NULL),(3,2,NULL,'a_condition_entry',0,NULL,'ACTIVE',NULL,1,NULL,NULL,'a_type_single_entry',NULL),(4,3,6,'a_less_six_months',0,NULL,'ACTIVE',NULL,1,'0-6 months',NULL,'a_type_multiple_choice',NULL),(5,3,7,'a_six_twelve_months',1,NULL,'ACTIVE',NULL,1,'6-123 months',NULL,'a_type_multiple_choice',NULL),(6,3,8,'a_one_twa_years',2,NULL,'ACTIVE',NULL,1,'1-2 years',NULL,'a_type_multiple_choice',NULL),(7,3,9,'a_twa_plus_years',3,NULL,'ACTIVE',NULL,1,'2+ years',NULL,'a_type_multiple_choice',NULL),(8,4,11,'a_yes_acne_worse',0,NULL,'ACTIVE',NULL,1,'Yes',NULL,'a_type_multiple_choice',NULL),(9,4,12,'a_na_acne_worse',1,NULL,'ACTIVE',NULL,1,'No',NULL,'a_type_multiple_choice',NULL),(12,7,17,'a_prescription_prev_treatment_type',0,NULL,'ACTIVE',NULL,1,'Prescription',NULL,'a_type_multiple_choice',NULL),(13,7,16,'a_otc_prev_treatment_type',1,NULL,'ACTIVE',NULL,1,'Over the counter',NULL,'a_type_multiple_choice',NULL),(14,7,18,'a_na_prev_treatment_type',2,NULL,'ACTIVE',NULL,1,'No treatments tried',NULL,'a_type_multiple_choice_none',NULL),(18,10,11,'a_yes_pregnancy_planning',0,372,'ACTIVE',1,1,'Yes','Pregnant, planning a pregnancy, or nursing','a_type_multiple_choice',NULL),(19,10,355,'a_na_pregnancy_planning',1,373,'ACTIVE',1,1,'No, I\'m not and will notify my doctor if I become pregnant during treatment','Not Pregnant','a_type_multiple_choice',NULL),(20,11,11,'a_yes_allergic_medications',0,NULL,'ACTIVE',NULL,1,'Yes',NULL,'a_type_multiple_choice',NULL),(21,11,12,'a_na_allergic_medications',1,NULL,'ACTIVE',NULL,1,'No',NULL,'a_type_multiple_choice',NULL),(24,14,35,'a_smoke_social_history',0,NULL,'ACTIVE',NULL,1,'Smoke tobacco',NULL,'a_type_multiple_choice',NULL),(25,14,36,'a_alcohol_social_history',1,NULL,'ACTIVE',NULL,1,'Drink alcohol',NULL,'a_type_multiple_choice',NULL),(26,14,37,'a_tanning_social_history',2,NULL,'ACTIVE',NULL,1,'Use tanning beds or sunbath',NULL,'a_type_multiple_choice',NULL),(27,15,11,'a_yes_prev_skin_diagnosis',0,NULL,'ACTIVE',NULL,1,'Yes',NULL,'a_type_multiple_choice',NULL),(28,15,12,'a_na_prev_skin_diagnosis',1,NULL,'ACTIVE',NULL,1,'No',NULL,'a_type_multiple_choice',NULL),(29,17,39,'a_alopecia_skin_diagnosis',0,NULL,'ACTIVE',NULL,1,'Alopecia (hair loss)',NULL,'a_type_multiple_choice',NULL),(30,17,40,'a_acne_skin_diagnosis',1,NULL,'ACTIVE',NULL,1,'Acne',NULL,'a_type_multiple_choice',NULL),(31,17,41,'a_eczema_skin_diagnosis',2,NULL,'ACTIVE',NULL,1,'Eczema',NULL,'a_type_multiple_choice',NULL),(32,17,42,'a_psoriasis_skin_diagnosis',3,NULL,'ACTIVE',NULL,1,'Psoriasis',NULL,'a_type_multiple_choice',NULL),(33,17,43,'a_rosacea_skin_diagnosis',4,NULL,'ACTIVE',NULL,1,'Rosacea',NULL,'a_type_multiple_choice',NULL),(34,17,44,'a_skin_cancer_diagnosis',5,NULL,'ACTIVE',NULL,1,'Skin cancer',NULL,'a_type_multiple_choice',NULL),(35,17,45,'a_other_skin_iagnosis',6,NULL,'ACTIVE',NULL,1,'Other',NULL,'a_type_multiple_choice_other_free_text',NULL),(36,16,48,'a_arthritis_diagnosis',0,NULL,'ACTIVE',NULL,1,'Arthritis',NULL,'a_type_multiple_choice',NULL),(37,16,49,'a_heart_valve_diagnosis',1,NULL,'ACTIVE',NULL,1,'Artifical Heart Valve',NULL,'a_type_multiple_choice',NULL),(38,16,50,'a_artificial_join__diagnosis',2,NULL,'ACTIVE',NULL,1,'Artifical Joint',NULL,'a_type_multiple_choice',NULL),(39,16,51,'a_asthma_diagnosis',3,NULL,'ACTIVE',NULL,1,'Asthma',NULL,'a_type_multiple_choice',NULL),(40,16,52,'a_blood_clots_diagnosis',4,NULL,'ACTIVE',NULL,1,'Blood Clots',NULL,'a_type_multiple_choice',NULL),(41,16,53,'a_diabetes_diagnosis',5,NULL,'ACTIVE',NULL,1,'Diabetes',NULL,'a_type_multiple_choice',NULL),(42,16,54,'a_epilepsey_diagnosis',6,NULL,'ACTIVE',NULL,1,'Epilepsy or Seizures',NULL,'a_type_multiple_choice',NULL),(43,16,55,'a_high_blood_pressure_diagnosis',7,NULL,'ACTIVE',NULL,1,'High blood pressure',NULL,'a_type_multiple_choice',NULL),(44,16,56,'a_high_cholestrol_diagnosis',8,NULL,'ACTIVE',NULL,1,'High Cholestrol',NULL,'a_type_multiple_choice',NULL),(45,16,57,'a_hiv_diagnosis',9,NULL,'ACTIVE',NULL,1,'HIV/AIDs',NULL,'a_type_multiple_choice',NULL),(46,16,58,'a_heart_attack_diagnosis',10,NULL,'ACTIVE',NULL,1,'Heart Attack',NULL,'a_type_multiple_choice',NULL),(47,16,59,'a_heart_murmur_diagnosis',11,NULL,'ACTIVE',NULL,1,'Heart Murmur',NULL,'a_type_multiple_choice',NULL),(48,16,60,'a_irregular_heart_beat_skin_diagnosis',12,NULL,'ACTIVE',NULL,1,'Irregular Heartbeat',NULL,'a_type_multiple_choice',NULL),(49,16,61,'a_kidney_disease_diagnosis',13,NULL,'ACTIVE',NULL,1,'Kidney Disease',NULL,'a_type_multiple_choice',NULL),(50,16,62,'a_liver_disease_diagnosis',14,NULL,'ACTIVE',NULL,1,'Liver disease',NULL,'a_type_multiple_choice',NULL),(51,16,63,'a_lung_disease_diagnosis',15,NULL,'ACTIVE',NULL,1,'Lung Disease',NULL,'a_type_multiple_choice',NULL),(52,16,64,'a_lupus_disease_diagnosis',16,NULL,'ACTIVE',NULL,1,'Lupus',NULL,'a_type_multiple_choice',NULL),(53,16,65,'a_organ_transplant_diagnosis',17,NULL,'ACTIVE',NULL,1,'Organ Transplant',NULL,'a_type_multiple_choice',NULL),(55,16,66,'a_pacemaker_diagnosis',18,NULL,'ACTIVE',NULL,1,'Pacemaker',NULL,'a_type_multiple_choice',NULL),(56,16,67,'a_thyroid_diagnosis',19,NULL,'ACTIVE',NULL,1,'Thyroid Problems',NULL,'a_type_multiple_choice',NULL),(57,16,68,'a_other_skin_diagnosis',20,NULL,'ACTIVE',NULL,1,'Other Condition Not Listed',NULL,'a_type_multiple_choice',NULL),(58,16,69,'a_none_skin_diagnosis',21,NULL,'ACTIVE',NULL,1,'No past or present conditions',NULL,'a_type_multiple_choice',NULL),(59,18,71,'a_face_acne_location',4,NULL,'ACTIVE',NULL,1,'Face',NULL,'a_type_multiple_choice',NULL),(60,18,72,'a_chest_acne_location',6,NULL,'ACTIVE',NULL,1,'Chest',NULL,'a_type_multiple_choice',NULL),(61,18,73,'a_back_acne_location',7,NULL,'ACTIVE',NULL,1,'Back',NULL,'a_type_multiple_choice',NULL),(62,18,74,'a_other_acne_location',8,NULL,'INACTIVE',NULL,1,'Other',NULL,'a_type_multiple_choice',NULL),(63,19,82,'a_face_front_phota_intake',0,NULL,'ACTIVE',NULL,1,'Face Front',NULL,'a_type_photo_entry_face_middle',NULL),(64,48,84,'a_face_right_photo_intake',1,NULL,'ACTIVE',NULL,1,'Profile Right',NULL,'a_type_photo_entry_face_right',NULL),(65,47,83,'a_face_left_photo_intake',2,NULL,'ACTIVE',NULL,1,'Profile Left',NULL,'a_type_photo_entry_face_left',NULL),(66,20,85,'a_chest_phota_intake',0,NULL,'ACTIVE',NULL,1,'Chest',NULL,'a_type_photo_entry_chest',NULL),(68,21,73,'a_back_phota_intake',0,NULL,'ACTIVE',NULL,1,'Back',NULL,'a_type_photo_entry_back',NULL),(69,22,109,'a_other_phota_intake',0,NULL,'ACTIVE',NULL,1,'Other',NULL,'a_type_photo_entry_other',NULL),(70,24,111,'a_effective_treatment_not_very',0,178,'ACTIVE',NULL,1,'Not Very','Not very effective','a_type_segmented_control',NULL),(71,24,112,'a_effective_treatment_somewhat',1,179,'ACTIVE',NULL,1,'Somewhat','Somewhat effective','a_type_segmented_control',NULL),(72,24,113,'a_effective_treatment_very',2,180,'ACTIVE',NULL,1,'Very','Very effective','a_type_segmented_control',NULL),(73,25,11,'a_using_treatment_yes',0,182,'ACTIVE',NULL,1,'Yes','Currently using it','a_type_segmented_control',NULL),(75,25,12,'a_using_treatment_no',1,181,'ACTIVE',NULL,1,'No','Not currently using it','a_type_segmented_control',NULL),(76,26,115,'a_length_treatment_less_one',0,183,'ACTIVE',NULL,1,'0-1','Used for less than one month','a_type_segmented_control',NULL),(77,26,116,'a_length_treatment_two_five_months',1,184,'ACTIVE',NULL,1,'2-5','Used for 2-5 months','a_type_segmented_control',NULL),(78,26,117,'a_length_treatment_six_eleven_months',2,185,'ACTIVE',NULL,1,'6-11','Used for 6-11 months','a_type_segmented_control',NULL),(79,26,118,'a_length_treatment_twelve_plus_months',3,186,'ACTIVE',NULL,1,'12+','Used for 12+ months','a_type_segmented_control',NULL),(80,28,151,'a_puberty',0,NULL,'INACTIVE',NULL,1,'During puberty',NULL,'a_type_multiple_choice',NULL),(81,28,152,'a_onset_six_months',4,NULL,'ACTIVE',NULL,1,'0-6 months ago',NULL,'a_type_multiple_choice',NULL),(82,28,153,'a_onset_one_two_years',6,NULL,'ACTIVE',NULL,1,'1-2 years ago',NULL,'a_type_multiple_choice',NULL),(83,28,154,'a_onset_more_two_years',7,NULL,'ACTIVE',NULL,1,'2 or more years ago',NULL,'a_type_multiple_choice',NULL),(84,29,157,'a_painful_touch',18,NULL,'ACTIVE',NULL,1,'Been painful to the touch',NULL,'a_type_multiple_choice',NULL),(85,29,158,'a_scarring',16,NULL,'INACTIVE',NULL,1,'Scarring',NULL,'a_type_multiple_choice',NULL),(86,29,159,'a_discoloration',20,NULL,'ACTIVE',NULL,1,'Caused discoloration',NULL,'a_type_multiple_choice',NULL),(87,30,11,'a_acne_worse_yes',0,NULL,'ACTIVE',NULL,1,'Yes',NULL,'a_type_multiple_choice',NULL),(88,30,12,'a_acne_worse_no',1,NULL,'ACTIVE',NULL,1,'No',NULL,'a_type_multiple_choice',NULL),(89,31,11,'a_periods_regular_yes',0,NULL,'ACTIVE',NULL,1,'Yes',NULL,'a_type_multiple_choice',NULL),(90,31,12,'a_periods_regular_no',1,NULL,'ACTIVE',NULL,1,'No',NULL,'a_type_multiple_choice',NULL),(91,32,165,'a_normal_skin',6,NULL,'ACTIVE',NULL,1,'Normal',NULL,'a_type_multiple_choice',NULL),(92,32,166,'a_oil_skin',7,NULL,'ACTIVE',NULL,1,'Oily',NULL,'a_type_multiple_choice',NULL),(93,32,167,'a_dry_skin',9,NULL,'ACTIVE',NULL,1,'Dry',NULL,'a_type_multiple_choice',NULL),(94,32,168,'a_combination_skin',8,NULL,'ACTIVE',NULL,1,'Combination',NULL,'a_type_multiple_choice',NULL),(95,33,11,'a_topical_allergic_medication_yes',0,NULL,'ACTIVE',NULL,1,'Yes',NULL,'a_type_multiple_choice',NULL),(96,33,12,'a_topical_allergic_medication_no',1,NULL,'ACTIVE',NULL,1,'No',NULL,'a_type_multiple_choice',NULL),(97,34,174,'a_other_condition_acne_gastiris',25,NULL,'ACTIVE',NULL,1,'Gastritis',NULL,'a_type_multiple_choice',NULL),(98,34,175,'a_other_condition_acne_colitis',8,NULL,'INACTIVE',NULL,1,'Colitis',NULL,'a_type_multiple_choice',NULL),(99,34,176,'a_other_condition_acne_kidney_condition',28,NULL,'ACTIVE',NULL,1,'Kidney disease',NULL,'a_type_multiple_choice',NULL),(100,34,177,'a_other_condition_acne_lupus',30,NULL,'ACTIVE',NULL,1,'Lupus',NULL,'a_type_multiple_choice',NULL),(102,37,196,'a_doctor_acne_vulgaris',3,332,'ACTIVE',NULL,1,'Acne vulgaris','acne','a_type_multiple_choice',NULL),(103,37,197,'a_doctor_acne_rosacea',4,197,'ACTIVE',NULL,1,'Acne rosacea','Acne rosacea','a_type_multiple_choice',NULL),(104,37,3,'a_doctor_acne_something_else',6,NULL,'ACTIVE',NULL,1,'Other',NULL,'a_type_multiple_choice',NULL),(105,38,199,'a_doctor_acne_severity_mild',0,199,'ACTIVE',NULL,1,'Mild','Mild','a_type_multiple_choice',NULL),(106,38,200,'a_doctor_acne_severity_moderate',1,200,'ACTIVE',NULL,1,'Moderate','Moderate','a_type_multiple_choice',NULL),(107,38,201,'a_doctor_acne_severity_severity',2,201,'ACTIVE',NULL,1,'Severe','Severe','a_type_multiple_choice',NULL),(108,39,203,'a_acne_whiteheads',0,NULL,'INACTIVE',NULL,1,'Whiteheads',NULL,'a_type_multiple_choice',NULL),(109,39,204,'a_acne_pustules',1,NULL,'INACTIVE',NULL,1,'Pustules',NULL,'a_type_multiple_choice',NULL),(110,39,205,'a_acne_nodules',2,NULL,'INACTIVE',NULL,1,'Nodules',NULL,'a_type_multiple_choice',NULL),(111,39,206,'a_acne_inflammatory',9,206,'ACTIVE',NULL,1,'Inflammatory','Inflammatory','a_type_multiple_choice',NULL),(112,39,207,'a_acne_blackheads',4,NULL,'INACTIVE',NULL,1,'Blackheads',NULL,'a_type_multiple_choice',NULL),(113,39,208,'a_acne_papules',5,NULL,'INACTIVE',NULL,1,'Papules',NULL,'a_type_multiple_choice',NULL),(114,39,209,'a_acne_cysts',10,209,'ACTIVE',NULL,1,'Cystic','Cystic','a_type_multiple_choice',NULL),(115,39,210,'a_acne_hormonal',11,210,'ACTIVE',NULL,1,'Hormonal','Hormonal','a_type_multiple_choice',NULL),(116,29,297,'a_cysts',12,NULL,'INACTIVE',NULL,1,'Turned into cysts',NULL,'a_type_multiple_choice',NULL),(117,29,298,'a_symptoms_none',22,NULL,'ACTIVE',NULL,1,'None of the above',NULL,'a_type_multiple_choice_none',NULL),(118,40,11,'a_irritate_skin_yes',0,300,'ACTIVE',NULL,1,'Yes','Irritated skin','a_type_segmented_control',NULL),(119,40,12,'a_irritate_skin_no',1,301,'ACTIVE',NULL,1,'No','Did not irritate skin','a_type_segmented_control',NULL),(120,10,302,'a_pregnant',2,321,'INACTIVE',1,1,'Pregnant','Currently pregnant','a_type_multiple_choice',NULL),(121,10,303,'a_nursing',3,322,'INACTIVE',1,1,'Nursing','Currently nursing','a_type_multiple_choice',NULL),(122,10,304,'a_planning_pregnancy',4,323,'INACTIVE',1,1,'Planning a pregnancy','Currently planning a pregnancy','a_type_multiple_choice',NULL),(123,10,305,'a_planning_pregnancy_none',5,324,'INACTIVE',NULL,1,'None of the above','Not currently pregnant, planning a pregnancy or nursing','a_type_multiple_choice_none',NULL),(124,41,115,'a_length_current_medication_less_than_month',0,307,'ACTIVE',NULL,1,'0-1','Taken for less than 1 month','a_type_segmented_control',NULL),(125,41,116,'a_length_current_medication_two_five_months',1,308,'ACTIVE',NULL,1,'2-5','Taken for 2-5 months','a_type_segmented_control',NULL),(126,41,117,'a_length_current_medication_six_eleven_months',2,309,'ACTIVE',NULL,1,'6-11','Taken for 6-11 months','a_type_segmented_control',NULL),(127,41,118,'a_length_current_medication_twelve_plus_months',3,310,'ACTIVE',NULL,1,'12+','Taken for 12+ months','a_type_segmented_control',NULL),(128,34,311,'a_other_condition_acne_hypertension',10,NULL,'INACTIVE',NULL,1,'Hypertension',NULL,'a_type_multiple_choice',NULL),(129,34,312,'a_other_condition_acne_polycystic_ovary_syndrome',32,NULL,'ACTIVE',NULL,1,'Polycystic ovary syndrome',NULL,'a_type_multiple_choice',NULL),(130,34,298,'a_other_condition_acne_none',33,NULL,'ACTIVE',NULL,1,'None of the above',NULL,'a_type_multiple_choice_none',NULL),(131,34,62,'a_other_condition_acne_liver_disease',29,NULL,'ACTIVE',NULL,1,'Liver disease',NULL,'a_type_multiple_choice',NULL),(132,18,314,'a_neck_acne_location',5,NULL,'INACTIVE',NULL,1,'Neck',NULL,'a_type_multiple_choice',NULL),(133,42,314,'a_neck_photo_intake',0,NULL,'ACTIVE',NULL,1,'Neck',NULL,'a_type_photo_entry_neck',NULL),(134,43,NULL,'a_other_acne_location_entry',0,NULL,'ACTIVE',NULL,1,NULL,NULL,'a_type_single_entry',NULL),(135,44,NULL,'a_other_skin_condition_entry',0,NULL,'ACTIVE',NULL,1,NULL,NULL,'a_type_single_entry',NULL),(136,37,326,'a_doctor_acne_perioral_dermatitis',5,326,'ACTIVE',NULL,1,'Perioral dermatitis','Perioral dermatitis','a_type_multiple_choice',NULL),(137,39,327,'a_acne_comedonal',8,327,'ACTIVE',NULL,1,'Comedonal','Comedonal','a_type_multiple_choice',NULL),(138,45,328,'a_acne_erythematotelangiectatic_rosacea',0,328,'ACTIVE',NULL,1,'Erythematotelangiectatic','Erythematotelangiectatic','a_type_multiple_choice',NULL),(139,45,329,'a_acne_papulopstular_rosacea',1,329,'ACTIVE',NULL,1,'Papulopustular','Papulopustular','a_type_multiple_choice',NULL),(140,45,330,'a_acne_rhinophyma_rosacea',2,330,'ACTIVE',NULL,1,'Rhinophyma','Rhinophyma','a_type_multiple_choice',NULL),(141,45,331,'a_acne_ocular_rosacea',3,331,'ACTIVE',NULL,1,'Ocular','Ocular','a_type_multiple_choice',NULL),(142,28,333,'a_six_twelve_months_ago',5,333,'ACTIVE',NULL,1,'6-12 months ago','6-12 months ago','a_type_multiple_choice',NULL),(143,46,11,'a_current_medications_yes',0,NULL,'ACTIVE',NULL,1,'Yes',NULL,'a_type_multiple_choice',NULL),(144,46,12,'a_current_medications_no',1,336,'ACTIVE',NULL,1,'No','None','a_type_multiple_choice',NULL),(145,49,348,'a_generic_only',0,NULL,'ACTIVE',1,1,'Generic only',NULL,'a_type_multiple_choice',NULL),(146,49,349,'a_no_preference',1,NULL,'ACTIVE',0,1,'No preference',NULL,'a_type_multiple_choice',NULL),(147,34,55,'a_other_condition_acne_high_bp',26,NULL,'ACTIVE',NULL,1,'High blood pressure',NULL,'a_type_multiple_choice',NULL),(148,34,352,'a_other_condition_acne_intestinal_inflammation',27,NULL,'ACTIVE',NULL,1,'Intestinal inflammation',NULL,'a_type_multiple_choice',NULL),(149,34,353,'a_other_condition_acne_organ_transplant',31,NULL,'ACTIVE',NULL,1,'Organ transplant',NULL,'a_type_multiple_choice',NULL),(150,29,356,'a_picked_or_squeezed',17,NULL,'ACTIVE',0,1,'Been picked or squeezed',NULL,'a_type_multiple_choice',NULL),(151,29,357,'a_created_scars',21,NULL,'ACTIVE',0,1,'Created scars',NULL,'a_type_multiple_choice',NULL),(152,50,11,'a_acne_prev_prescriptions_yes',0,NULL,'ACTIVE',0,1,'Yes',NULL,'a_type_multiple_choice',NULL),(153,50,12,'a_acne_prev_prescriptions_no',1,NULL,'ACTIVE',0,1,'No',NULL,'a_type_multiple_choice',NULL),(154,51,11,'a_acne_prev_otc_treatments_yes',0,NULL,'ACTIVE',0,1,'Yes',NULL,'a_type_multiple_choice',NULL),(155,51,12,'a_acne_prev_otc_treatments_no',1,NULL,'ACTIVE',0,1,'No',NULL,'a_type_multiple_choice',NULL),(156,53,11,'a_using_otc_yes',0,182,'ACTIVE',NULL,1,'Yes','Currently using it','a_type_segmented_control',NULL),(157,53,12,'a_using_otc_no',1,181,'ACTIVE',NULL,1,'No','Not currently using it','a_type_segmented_control',NULL),(158,54,111,'a_effective_otc_not_very',0,178,'ACTIVE',NULL,1,'Not Very','Not very effective','a_type_segmented_control',NULL),(159,54,112,'a_effective_otc_somewhat',1,179,'ACTIVE',NULL,1,'Somewhat','Somewhat effective','a_type_segmented_control',NULL),(160,54,113,'a_effective_otc_very',2,180,'ACTIVE',NULL,1,'Very','Very effective','a_type_segmented_control',NULL),(161,55,11,'a_otc_irritate_skin_yes',0,300,'ACTIVE',NULL,1,'Yes','Irritated skin','a_type_segmented_control',NULL),(162,55,12,'a_otc_irritate_skin_no',1,301,'ACTIVE',NULL,1,'No','Did not irritate skin','a_type_segmented_control',NULL),(163,56,115,'a_length_otc_less_one',0,183,'ACTIVE',NULL,1,'0-1','Used for less than one month','a_type_segmented_control',NULL),(164,56,116,'a_length_otc_two_five_months',1,184,'ACTIVE',NULL,1,'2-5','Used for 2-5 months','a_type_segmented_control',NULL),(165,56,117,'a_length_otc_two_six_eleven_months',2,185,'ACTIVE',NULL,1,'6-11','Used for 6-11 months','a_type_segmented_control',NULL),(166,56,118,'a_length_otc_twelve_plus_months',3,186,'ACTIVE',NULL,1,'12+','Used for 12+ months','a_type_segmented_control',NULL),(167,37,374,'a_doctor_acne_not_suitable_spruce',7,NULL,'ACTIVE',NULL,1,'Not Suitable For Spruce',NULL,'a_type_multiple_choice',NULL),(168,63,382,'a_benzaclin',11,NULL,'ACTIVE',NULL,1,'BenzaClin',NULL,'a_type_multiple_choice',NULL),(169,63,383,'a_benzoyl_peroxide',12,NULL,'ACTIVE',NULL,1,'Benzoyl Peroxide',NULL,'a_type_multiple_choice',NULL),(170,63,384,'a_clindamycin',13,NULL,'ACTIVE',NULL,1,'Clindamycin',NULL,'a_type_multiple_choice',NULL),(171,63,385,'a_differin',14,NULL,'ACTIVE',NULL,1,'Differin',NULL,'a_type_multiple_choice',NULL),(172,63,386,'a_duac',16,NULL,'ACTIVE',NULL,1,'Duac',NULL,'a_type_multiple_choice',NULL),(173,63,387,'a_epiduo',17,NULL,'ACTIVE',NULL,1,'Epiduo',NULL,'a_type_multiple_choice',NULL),(174,63,388,'a_metrogel',18,NULL,'ACTIVE',NULL,1,'Metrogel',NULL,'a_type_multiple_choice',NULL),(175,63,389,'a_minocycline',19,NULL,'ACTIVE',NULL,1,'Minocycline',NULL,'a_type_multiple_choice',NULL),(176,63,390,'a_retina_or_tretinoin',20,NULL,'ACTIVE',NULL,1,'Retin-A or Tretinoin',NULL,'a_type_multiple_choice',NULL),(177,63,391,'a_tetracycline',21,NULL,'ACTIVE',NULL,1,'Tetracycline',NULL,'a_type_multiple_choice',NULL),(178,63,109,'a_other_prev_acne_prescription',22,NULL,'ACTIVE',NULL,1,'Other',NULL,'a_type_multiple_choice_other_free_text',NULL),(179,64,11,'a_using_prev_prescription_yes',0,122,'ACTIVE',NULL,1,'Yes','Currently using it','a_type_segmented_control',NULL),(180,64,12,'a_using_prev_prescription_no',1,123,'ACTIVE',NULL,1,'No','Not currently using it','a_type_segmented_control',NULL),(181,65,394,'a_how_effective_prev_acne_prescription_not',0,395,'ACTIVE',NULL,1,'Not very','Not Effective','a_type_segmented_control',NULL),(182,65,120,'a_how_effective_prev_acne_prescription_somewhat',1,179,'ACTIVE',NULL,1,'Somewhat','Somewhat effective','a_type_segmented_control',NULL),(183,65,121,'a_how_effective_prev_acne_prescription_very_effective',2,180,'ACTIVE',NULL,1,'Very','Very effective','a_type_segmented_control',NULL),(184,66,11,'a_use_more_three_months_prev_acne_prescription_yes',0,397,'ACTIVE',NULL,1,'Yes','Used for more than 3 months','a_type_segmented_control',NULL),(185,66,12,'a_use_more_three_months_prev_acne_prescription_no',1,398,'ACTIVE',NULL,1,'No','Not used for more than 3 months','a_type_segmented_control',NULL),(186,67,11,'a_irritate_skin_prev_acne_prescription_yes',0,300,'ACTIVE',NULL,1,'Yes','Irritated skin','a_type_segmented_control',NULL),(187,67,12,'a_irritate_skin_prev_acne_prescription_no',1,301,'ACTIVE',NULL,1,'No','Did not irritate skin','a_type_segmented_control',NULL),(188,69,403,'a_acne_free',19,NULL,'ACTIVE',NULL,1,'Acne Free',NULL,'a_type_multiple_choice',NULL),(189,69,404,'a_cetaphil',21,NULL,'ACTIVE',NULL,1,'Cetaphil',NULL,'a_type_multiple_choice',NULL),(190,69,405,'a_clean_clear',22,NULL,'ACTIVE',NULL,1,'Clean and Clear',NULL,'a_type_multiple_choice',NULL),(191,69,406,'a_clearasil',23,NULL,'ACTIVE',NULL,1,'Clearasil',NULL,'a_type_multiple_choice',NULL),(192,69,407,'a_noxzema',25,NULL,'ACTIVE',NULL,1,'Noxzema',NULL,'a_type_multiple_choice',NULL),(193,69,408,'a_oxy',26,NULL,'ACTIVE',NULL,1,'Oxy',NULL,'a_type_multiple_choice',NULL),(194,69,409,'a_proactiv',28,NULL,'ACTIVE',NULL,1,'Proactiv',NULL,'a_type_multiple_choice',NULL),(195,69,410,'a_zeno',7,NULL,'INACTIVE',NULL,1,'Zeno',NULL,'a_type_multiple_choice',NULL),(196,69,109,'a_other_prev_acne_otc',29,NULL,'ACTIVE',NULL,1,'Other',NULL,'a_type_multiple_choice_other_free_text',NULL),(197,71,11,'a_using_prev_otc_yes',0,122,'ACTIVE',NULL,1,'Yes','Currently using it','a_type_segmented_control',NULL),(198,71,12,'a_using_prev_otc_no',1,123,'ACTIVE',NULL,1,'No','Not currently using it','a_type_segmented_control',NULL),(199,72,394,'a_how_effective_prev_acne_otc_not',0,395,'ACTIVE',NULL,1,'Not very','Not Effective','a_type_segmented_control',NULL),(200,72,120,'a_how_effective_prev_acne_otc_somewhat',1,179,'ACTIVE',NULL,1,'Somewhat','Somewhat effective','a_type_segmented_control',NULL),(201,72,121,'a_how_effective_prev_acne_otc_very_effective',2,180,'ACTIVE',NULL,1,'Very','Very effective','a_type_segmented_control',NULL),(202,73,11,'a_irritate_skin_prev_acne_otc_yes',0,300,'ACTIVE',NULL,1,'Yes','Irritated skin','a_type_segmented_control',NULL),(203,73,12,'a_irritate_skin_prev_acne_otc_no',1,301,'ACTIVE',NULL,1,'No','Did not irritate skin','a_type_segmented_control',NULL),(204,69,421,'a_aveeno',20,NULL,'ACTIVE',NULL,1,'Aveeno',NULL,'a_type_multiple_choice',NULL),(205,69,422,'a_panoxyl',27,NULL,'ACTIVE',NULL,1,'PanOxyl',NULL,'a_type_multiple_choice',NULL),(206,63,423,'a_doxycycline',15,NULL,'ACTIVE',NULL,1,'Doxycycline',NULL,'a_type_multiple_choice',NULL),(207,75,425,'a_more_acne_blemishes_photo_comparison',0,426,'ACTIVE',NULL,1,'I usually have more acne blemishes.','Has more acne blemishes','a_type_multiple_choice',NULL),(208,75,427,'a_fewer_acne_blemishes_photo_comparison',1,428,'ACTIVE',NULL,1,'I usually have fewer acne blemishes.','Has fewer acne blemishes','a_type_multiple_choice',NULL),(209,75,429,'a_about_the_same_photo_comparison',2,430,'ACTIVE',NULL,1,'My skin usually looks about the same.','Looks about the same','a_type_multiple_choice',NULL),(210,76,433,'a_insurance_brand_generic',0,NULL,'ACTIVE',NULL,1,'Brand name and generic',NULL,'a_type_multiple_choice',NULL),(211,76,434,'a_insurance_generic_only',1,NULL,'ACTIVE',NULL,1,'Generic only',NULL,'a_type_multiple_choice',NULL),(212,76,435,'a_insurance_idk',2,439,'ACTIVE',NULL,1,'I don\'t know','Patient doesn\'t know','a_type_multiple_choice',NULL),(213,76,436,'a_no_insurance',3,438,'ACTIVE',NULL,1,'I don\'t have insurance','No insurance','a_type_multiple_choice',NULL),(214,32,440,'a_sensitive_skin',10,NULL,'ACTIVE',NULL,1,'Sensitive',NULL,'a_type_multiple_choice',NULL),(215,32,109,'a_other_skin',11,NULL,'ACTIVE',NULL,1,'Other',NULL,'a_type_multiple_choice_other_free_text',NULL),(216,29,442,'a_deep_lumps',19,NULL,'ACTIVE',NULL,1,'Formed deep, hard lumps',NULL,'a_type_multiple_choice',NULL),(217,77,445,'a_acne_worse_diet',0,NULL,'ACTIVE',NULL,1,'Diet',NULL,'a_type_multiple_choice',NULL),(218,77,446,'a_acne_worse_hair_products',1,NULL,'ACTIVE',NULL,1,'Hair products',NULL,'a_type_multiple_choice',NULL),(219,77,447,'a_acne_worse_makeup',2,NULL,'ACTIVE',NULL,1,'Makeup',NULL,'a_type_multiple_choice',NULL),(220,77,448,'a_acne_worse_hormonal_changes',3,NULL,'ACTIVE',NULL,1,'Hormonal changes',NULL,'a_type_multiple_choice',NULL),(221,77,449,'a_acne_worse_stress',4,NULL,'ACTIVE',NULL,1,'Stress',NULL,'a_type_multiple_choice',NULL),(222,77,450,'a_acne_worse_sweating_and_sports',5,NULL,'ACTIVE',NULL,1,'Sweating or sports',NULL,'a_type_multiple_choice',NULL),(223,77,451,'a_acne_worse_weater',6,NULL,'ACTIVE',NULL,1,'Weather',NULL,'a_type_multiple_choice',NULL),(224,77,452,'a_acne_worse_none_or_not_sure',7,453,'ACTIVE',NULL,1,'I\'m not sure','Unsure','a_type_multiple_choice',NULL),(225,77,109,'a_acne_worse_other',8,NULL,'ACTIVE',NULL,1,'Other',NULL,'a_type_multiple_choice_other_free_text',NULL),(226,69,455,'a_neutrogena',24,NULL,'ACTIVE',NULL,1,'Neutrogena',NULL,'a_type_multiple_choice',NULL),(227,78,459,'a_skin_improvements_very_happy',0,NULL,'ACTIVE',NULL,1,'Very happy',NULL,'a_type_multiple_choice',NULL),(228,78,460,'a_skin_improvements_happy',1,NULL,'ACTIVE',NULL,1,'Happy',NULL,'a_type_multiple_choice',NULL),(229,78,461,'a_skin_improvements_neutral',2,NULL,'ACTIVE',NULL,1,'Neutral',NULL,'a_type_multiple_choice',NULL),(230,78,462,'a_skin_improvements_unhappy',3,NULL,'ACTIVE',NULL,1,'Unhappy',NULL,'a_type_multiple_choice',NULL),(231,78,463,'a_skin_improvements_very_unhappy',4,NULL,'ACTIVE',NULL,1,'Very Unhappy',NULL,'a_type_multiple_choice',NULL),(232,80,470,'a_using_tp_as_instructed_yes',0,473,'ACTIVE',NULL,1,'Yes, completely','Compliant','a_type_multiple_choice',NULL),(233,80,471,'a_using_tp_as_instructed_mostly',1,474,'ACTIVE',NULL,1,'Mostly','Mostly compliant','a_type_multiple_choice',NULL),(234,80,500,'a_using_tp_as_instructed_sometimes',2,475,'ACTIVE',NULL,1,'sometimes','Somewhat compliant','a_type_multiple_choice',NULL),(235,80,12,'a_using_tp_as_instructed_no',3,476,'ACTIVE',NULL,1,'No','Not compliant','a_type_multiple_choice',NULL),(236,80,472,'a_using_tp_as_instructed_not_sure',4,477,'ACTIVE',NULL,1,'I\'m not sure','Not sure','a_type_multiple_choice',NULL),(237,81,11,'a_side_effects_from_tp_yes',0,NULL,'ACTIVE',NULL,1,'Yes',NULL,'a_type_multiple_choice',NULL),(238,81,12,'a_side_effects_from_tp_no',1,NULL,'ACTIVE',NULL,1,'No',NULL,'a_type_multiple_choice',NULL),(239,83,11,'a_using_all_treatments_in_tp_yes',0,NULL,'ACTIVE',NULL,1,'Yes',NULL,'a_type_multiple_choice',NULL),(240,83,12,'a_using_all_treatments_in_tp_no',1,NULL,'ACTIVE',NULL,1,'No',NULL,'a_type_multiple_choice',NULL),(241,86,11,'a_other_medications_since_tp_yes',0,NULL,'ACTIVE',NULL,1,'Yes',NULL,'a_type_multiple_choice',NULL),(242,86,12,'a_other_medications_since_tp_no',1,NULL,'ACTIVE',NULL,1,'No',NULL,'a_type_multiple_choice',NULL),(243,88,11,'a_medication_allergies_since_visit_yes',0,NULL,'ACTIVE',NULL,1,'Yes',NULL,'a_type_multiple_choice',NULL),(244,88,12,'a_medication_allergies_since_visit_no',1,NULL,'ACTIVE',NULL,1,'No',NULL,'a_type_multiple_choice',NULL),(245,90,11,'a_med_hx_changes_relevant_yes',0,NULL,'ACTIVE',NULL,1,'Yes',NULL,'a_type_multiple_choice',NULL),(246,90,12,'a_med_hx_changes_relevant_no',1,NULL,'ACTIVE',NULL,1,'No',NULL,'a_type_multiple_choice',NULL),(247,92,502,'a_diagnosis_severity_mild',0,NULL,'ACTIVE',NULL,1,'Mild',NULL,'a_type_multiple_choice',NULL),(248,92,503,'a_diagnosis_severity_moderate',1,NULL,'ACTIVE',NULL,1,'Moderate',NULL,'a_type_multiple_choice',NULL),(249,92,504,'a_diagnosis_severity_severe',2,NULL,'ACTIVE',NULL,1,'Severe',NULL,'a_type_multiple_choice',NULL),(250,93,327,'a_diagnosis_acne_vulgaris_type_comedonal',0,NULL,'ACTIVE',NULL,1,'Comedonal',NULL,'a_type_multiple_choice',NULL),(251,93,206,'a_diagnosis_acne_vulgaris_type_inflammatory',1,NULL,'ACTIVE',NULL,1,'Inflammatory',NULL,'a_type_multiple_choice',NULL),(252,93,506,'a_diagnosis_acne_vulgaris_type_cystic',2,NULL,'ACTIVE',NULL,1,'Cystic',NULL,'a_type_multiple_choice',NULL),(253,93,507,'a_diagnosis_acne_vulgaris_type_hormonal',3,NULL,'ACTIVE',NULL,1,'Hormonal',NULL,'a_type_multiple_choice',NULL),(254,94,328,'a_diagnosis_acne_rosacea_type_erythematotelangiectatic',0,NULL,'ACTIVE',NULL,1,'Erythematotelangiectatic',NULL,'a_type_multiple_choice',NULL),(255,94,329,'a_diagnosis_acne_rosacea_type_papulopstular',1,NULL,'ACTIVE',NULL,1,'Papulopustular',NULL,'a_type_multiple_choice',NULL),(256,94,330,'a_diagnosis_acne_rosacea_type_rhinophyma',2,NULL,'ACTIVE',NULL,1,'Rhinophyma',NULL,'a_type_multiple_choice',NULL),(257,94,331,'a_diagnosis_acne_rosacea_type_ocular',3,NULL,'ACTIVE',NULL,1,'Ocular',NULL,'a_type_multiple_choice',NULL);
/*!40000 ALTER TABLE `potential_answer` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `drug_route`
--

DROP TABLE IF EXISTS `drug_route`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `drug_route` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(150) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `drug_route` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `drug_route`
--

LOCK TABLES `drug_route` WRITE;
/*!40000 ALTER TABLE `drug_route` DISABLE KEYS */;
INSERT INTO `drug_route` VALUES (2,'compounding'),(4,'If this is'),(5,'If this is another'),(6,'If this is yet another'),(8,'injectable'),(11,'intravenous'),(7,'mucous membrane'),(10,'ophthalmic'),(3,'oral'),(9,'otic'),(1,'topical');
/*!40000 ALTER TABLE `drug_route` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `drug_form`
--

DROP TABLE IF EXISTS `drug_form`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `drug_form` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(150) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `drug_form` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=21 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `drug_form`
--

LOCK TABLES `drug_form` WRITE;
/*!40000 ALTER TABLE `drug_form` DISABLE KEYS */;
INSERT INTO `drug_form` VALUES (2,'bar'),(10,'capsule'),(3,'cream'),(15,'emulsion'),(4,'foam'),(5,'gel'),(6,'kit'),(7,'liquid'),(8,'lotion'),(9,'pad'),(1,'powder'),(19,'powder for injection'),(11,'Right'),(17,'soap'),(12,'solution'),(18,'spray'),(14,'suspension'),(20,'swab'),(13,'tablet'),(16,'tablet, chewable');
/*!40000 ALTER TABLE `drug_form` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `drug_supplemental_instruction`
--

DROP TABLE IF EXISTS `drug_supplemental_instruction`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `drug_supplemental_instruction` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `text` varchar(150) NOT NULL,
  `drug_name_id` int(10) unsigned NOT NULL,
  `drug_form_id` int(10) unsigned DEFAULT NULL,
  `drug_route_id` int(10) unsigned DEFAULT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `drug_name_id` (`drug_name_id`),
  KEY `drug_form_id` (`drug_form_id`),
  KEY `drug_route_id` (`drug_route_id`),
  CONSTRAINT `drug_supplemental_instruction_ibfk_1` FOREIGN KEY (`drug_name_id`) REFERENCES `drug_name` (`id`),
  CONSTRAINT `drug_supplemental_instruction_ibfk_2` FOREIGN KEY (`drug_form_id`) REFERENCES `drug_form` (`id`),
  CONSTRAINT `drug_supplemental_instruction_ibfk_3` FOREIGN KEY (`drug_route_id`) REFERENCES `drug_route` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=15 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `drug_supplemental_instruction`
--

LOCK TABLES `drug_supplemental_instruction` WRITE;
/*!40000 ALTER TABLE `drug_supplemental_instruction` DISABLE KEYS */;
INSERT INTO `drug_supplemental_instruction` VALUES (1,'Benzoyl peroxide level instruction 1',1,NULL,NULL,'INACTIVE','2013-12-28 19:14:40.000000'),(2,'Benzoyl peroxide level instruction 2',1,NULL,NULL,'INACTIVE','2013-12-28 19:14:40.000000'),(3,'Benzoyl peroxide and route topical level instruction 1',1,NULL,1,'INACTIVE','2013-12-28 19:14:40.000000'),(4,'Benzoyl peroxide and route compounding level instruction 1',1,NULL,2,'INACTIVE','2013-12-28 19:14:40.000000'),(5,'Benzoyl peroxide, route topical and form cream level instruction 1',1,3,1,'INACTIVE','2013-12-28 19:14:41.000000'),(6,'Benzoyl peroxide, route topical and form gel level instruction 1',1,5,1,'INACTIVE','2013-12-28 19:14:41.000000'),(7,'Benzoyl peroxide, route topical and form liquid level instruction 1',1,7,1,'INACTIVE','2013-12-28 19:14:41.000000'),(8,'Benzoyl peroxide level instruction 1',1,NULL,NULL,'ACTIVE','2013-12-30 13:30:58.000000'),(9,'Benzoyl peroxide level instruction 2',1,NULL,NULL,'ACTIVE','2013-12-30 13:30:58.000000'),(10,'Benzoyl peroxide and route topical level instruction 1',1,NULL,1,'ACTIVE','2013-12-30 13:30:58.000000'),(11,'Benzoyl peroxide and route compounding level instruction 1',1,NULL,2,'ACTIVE','2013-12-30 13:30:58.000000'),(12,'Benzoyl peroxide, route topical and form cream level instruction 1',1,3,1,'ACTIVE','2013-12-30 13:30:58.000000'),(13,'Benzoyl peroxide, route topical and form gel level instruction 1',1,5,1,'ACTIVE','2013-12-30 13:30:59.000000'),(14,'Benzoyl peroxide, route topical and form liquid level instruction 1',1,7,1,'ACTIVE','2013-12-30 13:30:59.000000');
/*!40000 ALTER TABLE `drug_supplemental_instruction` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `deny_refill_reason`
--

DROP TABLE IF EXISTS `deny_refill_reason`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `deny_refill_reason` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `reason_code` varchar(100) NOT NULL,
  `reason` varchar(150) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `deny_refill_reason`
--

LOCK TABLES `deny_refill_reason` WRITE;
/*!40000 ALTER TABLE `deny_refill_reason` DISABLE KEYS */;
INSERT INTO `deny_refill_reason` VALUES (1,'DeniedPatientUnknown','Patient unknown to the provider'),(2,'DeniedPatientNotUnderCare','Patient never under provider care'),(3,'DeniedPatientNoLongerUnderPatientCare','Patient no longer under provider care'),(4,'DeniedTooSoon','Refill too soon'),(5,'DeniedNeverPrescribed','Medication never prescribed for patient'),(6,'DeniedHavePatientContact','Patient should contact provider'),(7,'DeniedRefillInappropriate','Refill not appropriate'),(8,'DeniedAlreadyPickedUp','Patient has picked up prescription'),(9,'DeniedAlreadyPickedUpPartialFill','Patient has picked up partial fill of prescription'),(10,'DeniedNotPickedUp','Patient has not picked up prescription, drug returned to stock'),(11,'DeniedChangeInappropriate','Change not appropriate'),(12,'DeniedNeedAppointment','Patient needs appointment'),(13,'DeniedPrescriberNotAssociateWithLocation','Prescriber not associated with this practice or location'),(14,'DeniedNoPriorAuthAttempt','No attempt will be made to obtain Prior Authorization'),(15,'DeniedAlreadyHandled','Request already responded to by other means (e.g. phone or fax)'),(16,'DeniedNewRx','New RX to follow');
/*!40000 ALTER TABLE `deny_refill_reason` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `state`
--

DROP TABLE IF EXISTS `state`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `state` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `full_name` varchar(300) NOT NULL,
  `abbreviation` varchar(10) NOT NULL,
  `country` varchar(300) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=101 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `state`
--

LOCK TABLES `state` WRITE;
/*!40000 ALTER TABLE `state` DISABLE KEYS */;
INSERT INTO `state` VALUES (51,'Alabama','AL','USA'),(52,'Alaska','AK','USA'),(53,'Arizona','AZ','USA'),(54,'Arkansas','AR','USA'),(55,'California','CA','USA'),(56,'Colorado','CO','USA'),(57,'Connecticut','CT','USA'),(58,'Delaware','DE','USA'),(59,'Florida','FL','USA'),(60,'Georgia','GA','USA'),(61,'Hawaii','HI','USA'),(62,'Idaho','ID','USA'),(63,'Illinois','IL','USA'),(64,'Indiana','IN','USA'),(65,'Iowa','IA','USA'),(66,'Kansas','KS','USA'),(67,'Kentucky','KY','USA'),(68,'Louisiana','LA','USA'),(69,'Maine','ME','USA'),(70,'Maryland','MD','USA'),(71,'Massachusetts','MA','USA'),(72,'Michigan','MI','USA'),(73,'Minnesota','MN','USA'),(74,'Mississippi','MS','USA'),(75,'Missouri','MO','USA'),(76,'Montana','MT','USA'),(77,'Nebraska','NE','USA'),(78,'Nevada','NV','USA'),(79,'New Hampshire','NH','USA'),(80,'New Jersey','NJ','USA'),(81,'New Mexico','NM','USA'),(82,'New York','NY','USA'),(83,'North Carolina','NC','USA'),(84,'North Dakota','ND','USA'),(85,'Ohio','OH','USA'),(86,'Oklahoma','OK','USA'),(87,'Oregon','OR','USA'),(88,'Pennsylvania','PA','USA'),(89,'Rhode Island','RI','USA'),(90,'South Carolina','SC','USA'),(91,'South Dakota','SD','USA'),(92,'Tennessee','TN','USA'),(93,'Texas','TX','USA'),(94,'Utah','UT','USA'),(95,'Vermont','VT','USA'),(96,'Virginia','VA','USA'),(97,'Washington','WA','USA'),(98,'West Virginia','WV','USA'),(99,'Wisconsin','WI','USA'),(100,'Wyoming','WY','USA');
/*!40000 ALTER TABLE `state` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `photo_slot`
--

DROP TABLE IF EXISTS `photo_slot`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `photo_slot` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `question_id` int(10) unsigned NOT NULL,
  `required` tinyint(1) NOT NULL,
  `status` varchar(100) NOT NULL,
  `placeholder_image_tag` varchar(100) DEFAULT NULL,
  `ordering` int(10) unsigned NOT NULL,
  `language_id` int(10) unsigned DEFAULT '1',
  `name_text` varchar(600) DEFAULT NULL,
  `photo_slot_type` varchar(60) NOT NULL,
  `client_data` blob,
  PRIMARY KEY (`id`),
  KEY `question_id` (`question_id`),
  KEY `fk_photo_slot_languages_supported_id` (`language_id`),
  CONSTRAINT `fk_photo_slot_languages_supported_id` FOREIGN KEY (`language_id`) REFERENCES `languages_supported` (`id`),
  CONSTRAINT `photo_slot_ibfk_1` FOREIGN KEY (`question_id`) REFERENCES `question` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `photo_slot`
--

LOCK TABLES `photo_slot` WRITE;
/*!40000 ALTER TABLE `photo_slot` DISABLE KEYS */;
INSERT INTO `photo_slot` VALUES (1,59,1,'ACTIVE','photo_slot_face_front',0,1,'Face Front','photo_slot_face_front',NULL),(2,59,1,'ACTIVE','photo_slot_face_left',2,1,'Face Side','photo_slot_face_left',NULL),(3,59,1,'ACTIVE','photo_slot_face_right',1,1,'Face Side','photo_slot_face_right',NULL),(4,59,0,'ACTIVE','photo_slot_face_other',3,1,'Face','photo_slot_other',NULL),(5,61,1,'ACTIVE','photo_slot_back',0,1,'Back','photo_slot_back',NULL),(6,61,0,'ACTIVE','photo_slot_other',1,1,'Back','photo_slot_other',NULL),(7,60,1,'ACTIVE','photo_slot_chest',0,1,'Chest','photo_slot_chest',NULL),(8,60,0,'ACTIVE','photo_slot_other',1,1,'Chest','photo_slot_other',NULL),(9,62,1,'ACTIVE','photo_slot_other',0,1,'Other','photo_slot_other',NULL);
/*!40000 ALTER TABLE `photo_slot` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `photo_slot_type`
--

DROP TABLE IF EXISTS `photo_slot_type`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `photo_slot_type` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `slot_type` varchar(100) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `photo_slot_type`
--

LOCK TABLES `photo_slot_type` WRITE;
/*!40000 ALTER TABLE `photo_slot_type` DISABLE KEYS */;
INSERT INTO `photo_slot_type` VALUES (1,'photo_slot_face_front'),(2,'photo_slot_face_right'),(3,'photo_slot_face_left'),(4,'photo_slot_other'),(5,'photo_slot_back'),(6,'photo_slot_chest');
/*!40000 ALTER TABLE `photo_slot_type` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `role_type`
--

DROP TABLE IF EXISTS `role_type`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `role_type` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `role_type_tag` varchar(250) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `role_type`
--

LOCK TABLES `role_type` WRITE;
/*!40000 ALTER TABLE `role_type` DISABLE KEYS */;
INSERT INTO `role_type` VALUES (5,'ADMIN'),(6,'PATIENT'),(7,'DOCTOR'),(8,'MA');
/*!40000 ALTER TABLE `role_type` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `account_available_permission`
--

DROP TABLE IF EXISTS `account_available_permission`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `account_available_permission` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(60) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=25 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `account_available_permission`
--

LOCK TABLES `account_available_permission` WRITE;
/*!40000 ALTER TABLE `account_available_permission` DISABLE KEYS */;
INSERT INTO `account_available_permission` VALUES (2,'admin_accounts.edit'),(1,'admin_accounts.view'),(4,'analytics_reports.edit'),(3,'analytics_reports.view'),(22,'case.view'),(24,'cfg.edit'),(23,'cfg.view'),(7,'doctors.edit'),(8,'doctors.view'),(5,'email.edit'),(6,'email.view'),(21,'financial.view'),(20,'ftp.edit'),(19,'ftp.view'),(10,'layout.edit'),(9,'layout.view'),(12,'pathways.edit'),(11,'pathways.view'),(16,'resource_guides.edit'),(15,'resource_guides.view'),(18,'rx_guides.edit'),(17,'rx_guides.view'),(14,'stp.edit'),(13,'stp.view');
/*!40000 ALTER TABLE `account_available_permission` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `account_group`
--

DROP TABLE IF EXISTS `account_group`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `account_group` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(60) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `account_group`
--

LOCK TABLES `account_group` WRITE;
/*!40000 ALTER TABLE `account_group` DISABLE KEYS */;
INSERT INTO `account_group` VALUES (7,'case'),(8,'cfg'),(6,'financial'),(5,'ftp'),(2,'pathways'),(3,'resource_guides'),(4,'rx_guides'),(1,'superuser');
/*!40000 ALTER TABLE `account_group` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `account_group_permission`
--

DROP TABLE IF EXISTS `account_group_permission`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `account_group_permission` (
  `group_id` int(10) unsigned NOT NULL,
  `permission_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`group_id`,`permission_id`),
  KEY `permission_id` (`permission_id`),
  CONSTRAINT `account_group_permission_ibfk_1` FOREIGN KEY (`group_id`) REFERENCES `account_group` (`id`) ON DELETE CASCADE,
  CONSTRAINT `account_group_permission_ibfk_2` FOREIGN KEY (`permission_id`) REFERENCES `account_available_permission` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `account_group_permission`
--

LOCK TABLES `account_group_permission` WRITE;
/*!40000 ALTER TABLE `account_group_permission` DISABLE KEYS */;
INSERT INTO `account_group_permission` VALUES (1,1),(1,2),(1,3),(1,4),(1,5),(1,6),(1,7),(1,8),(1,9),(1,10),(1,11),(2,11),(1,12),(2,12),(1,13),(1,14),(1,15),(3,15),(1,16),(3,16),(1,17),(4,17),(1,18),(4,18),(1,19),(5,19),(1,20),(5,20),(1,21),(6,21),(1,22),(7,22),(1,23),(8,23),(1,24),(8,24);
/*!40000 ALTER TABLE `account_group_permission` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `email_sender`
--

DROP TABLE IF EXISTS `email_sender`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `email_sender` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `email` varchar(64) NOT NULL,
  `created` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `email_sender`
--

LOCK TABLES `email_sender` WRITE;
/*!40000 ALTER TABLE `email_sender` DISABLE KEYS */;
INSERT INTO `email_sender` VALUES (1,'Spruce Support','support@sprucehealth.com','2014-09-12 20:49:48','2014-09-12 20:49:48');
/*!40000 ALTER TABLE `email_sender` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sku_category`
--

DROP TABLE IF EXISTS `sku_category`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `sku_category` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `type` varchar(32) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `type` (`type`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sku_category`
--

LOCK TABLES `sku_category` WRITE;
/*!40000 ALTER TABLE `sku_category` DISABLE KEYS */;
INSERT INTO `sku_category` VALUES (3,'followup'),(2,'visit');
/*!40000 ALTER TABLE `sku_category` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sku`
--

DROP TABLE IF EXISTS `sku`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `sku` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `sku_category_id` int(10) unsigned NOT NULL,
  `type` varchar(128) NOT NULL,
  `status` varchar(32) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `type` (`type`),
  KEY `sku_category_id` (`sku_category_id`),
  CONSTRAINT `sku_ibfk_1` FOREIGN KEY (`sku_category_id`) REFERENCES `sku_category` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sku`
--

LOCK TABLES `sku` WRITE;
/*!40000 ALTER TABLE `sku` DISABLE KEYS */;
INSERT INTO `sku` VALUES (2,2,'acne_visit','ACTIVE'),(3,2,'acne_followup','ACTIVE'),(4,3,'derm_antiaging_and_skin_protection_followup','INACTIVE');
/*!40000 ALTER TABLE `sku` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `care_providing_state`
--

DROP TABLE IF EXISTS `care_providing_state`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `care_providing_state` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `state` varchar(100) NOT NULL,
  `long_state` varchar(250) NOT NULL,
  `clinical_pathway_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `clinical_pathway_id` (`clinical_pathway_id`),
  KEY `state` (`state`),
  CONSTRAINT `care_providing_state_ibfk_2` FOREIGN KEY (`clinical_pathway_id`) REFERENCES `clinical_pathway` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `care_providing_state`
--

LOCK TABLES `care_providing_state` WRITE;
/*!40000 ALTER TABLE `care_providing_state` DISABLE KEYS */;
INSERT INTO `care_providing_state` VALUES (1,'CA','California',1);
/*!40000 ALTER TABLE `care_providing_state` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `dispense_unit`
--

DROP TABLE IF EXISTS `dispense_unit`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dispense_unit` (
  `id` int(10) unsigned NOT NULL,
  `dispense_unit_text_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `dispense_unit_text_id` (`dispense_unit_text_id`),
  CONSTRAINT `dispense_unit_ibfk_1` FOREIGN KEY (`dispense_unit_text_id`) REFERENCES `app_text` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `dispense_unit`
--

LOCK TABLES `dispense_unit` WRITE;
/*!40000 ALTER TABLE `dispense_unit` DISABLE KEYS */;
INSERT INTO `dispense_unit` VALUES (1,212),(2,213),(3,214),(4,215),(5,216),(6,217),(7,218),(8,219),(9,220),(10,221),(11,222),(12,223),(13,224),(14,225),(15,226),(16,227),(18,229),(19,230),(20,231),(21,232),(22,233),(23,234),(24,235),(25,236),(26,237),(27,238),(28,239),(29,240),(30,241),(31,242),(32,243),(33,244),(34,245),(35,246),(36,247),(37,248),(38,249),(39,250),(40,251),(41,252),(42,253),(43,254),(44,255),(45,256),(46,257),(47,258),(48,259),(49,260),(50,261),(51,262),(52,263),(53,264),(54,265),(55,266),(56,267),(57,268),(58,269),(59,270),(60,271),(61,272),(62,273),(63,274),(64,275),(65,276),(66,277),(67,278),(68,279),(69,280),(70,281),(71,282),(72,283),(73,284),(74,285),(75,286),(76,287),(77,288),(78,289),(79,290),(80,291),(81,292),(82,293);
/*!40000 ALTER TABLE `dispense_unit` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `clinical_pathway`
--

DROP TABLE IF EXISTS `clinical_pathway`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `clinical_pathway` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `tag` varchar(64) NOT NULL,
  `name` varchar(250) NOT NULL,
  `medicine_branch` varchar(250) NOT NULL,
  `status` varchar(32) NOT NULL,
  `details_json` blob,
  `stp_json` blob,
  PRIMARY KEY (`id`),
  UNIQUE KEY `tag` (`tag`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `clinical_pathway`
--

LOCK TABLES `clinical_pathway` WRITE;
/*!40000 ALTER TABLE `clinical_pathway` DISABLE KEYS */;
INSERT INTO `clinical_pathway` VALUES (1,'health_condition_acne','Acne','Dermatology','ACTIVE','{\"what_is_included\":[\"Response from your doctor within 24 hours\",\"A personalized treatment plan\",\"30 days of follow-up messaging\"],\"who_will_treat_me\":\"Top board-certified dermatologists from across the U.S.\",\"right_for_me\":\"Common acne symptoms include whiteheads, blackheads, and red, inflamed patches of skin (such as cysts).\",\"did_you_know\":[\"You don’t need to let acne run its course. According to the American Academy of Dermatology, 99% of acne cases are treatable.\",\"Prescription treatments can eliminate 84-90% of acne lesions, while non-prescription treatments may eliminate only 16-34%.\",\"Acne is not a teenage problem. 50% of women in their 20s struggle with acne.\",\"95% of patients on Spruce saw substantial improvement in their skin within 12 weeks of their first acne visit.\"],\"faq\":[{\"question\":\"Do I have acne?\",\"answer\":\"Symptoms include whiteheads, blackheads, and red, inflamed patches of skin (such as cysts).\"},{\"question\":\"What if my skin is better today than it normally is?\",\"answer\":\"You’ll be able to tell your dermatologist if your skin is better or worse than normal and you can always message them with additional photos later.\"},{\"question\":\"Will I get a prescription?\",\"answer\":\"Your dermatologist will treat your case in a medically appropriate way, which may include prescriptions. However, prescriptions are not guaranteed, and if you are explicitly looking for Isotretinoin (accutane), you should see a dermatologist in person.\"},{\"question\":\"What if the doctor can\'t treat me?\",\"answer\":\"If your acne cannot be treated remotely, you may be referred to a dermatologist for an in-person appointment. The cost of your visit will be refunded.\"}]}','{\"header_view\": {\"type\": \"treatment:hero_header\", \"title\": \"Sample Treatment Plan\", \"subtitle\": \"Your doctor will personalize a treatment plan for you.\"}, \"treatment_views\": [{\"type\": \"treatment:card_view\", \"views\": [{\"type\": \"treatment:text\", \"text\": \"Your dermatologist will determine the right treatments for your skin.\", \"style\": \"title1_medium\"}, {\"type\": \"treatment:text\", \"text\": \"Prescriptions will be available to pick up at your preferred pharmacy.\"} ] }, {\"type\": \"treatment:card_view\", \"views\": [{\"type\": \"treatment:prescription\", \"small_header_text\": \"Prescription\", \"title\": \"Tretinoin Topical Cream 0.025%\", \"subtitle\": \"Prescription\", \"description\": \"Apply to the affected area every other night for the first two weeks. Then every night as tolerated.\", \"icon_url\": \"spruce:///images/icon_topical_prescription\", \"icon_width\": 50, \"icon_height\": 50, \"buttons\": [{\"type\": \"treatment:prescription_button\", \"text\": \"Prescription Guide\", \"icon_url\": \"spruce:///images/icon_rx_guide\", \"tap_url\": \"spruce:///action/view_treatment_guide?treatment_id=tretinoin\"} ] } ] }, {\"type\": \"treatment:card_view\", \"views\": [{\"type\": \"treatment:prescription\", \"subtitle\": \"Prescription\", \"title\": \"Doxycycline Hyclate 100mg Tablet\", \"description\": \"Take twice daily with small amount of food. Remain upright for 30 minutes after taking.\", \"icon_url\": \"spruce:///images/icon_oral_prescription\", \"icon_width\": 50, \"icon_height\": 50, \"buttons\": [{\"type\": \"treatment:prescription_button\", \"text\": \"Prescription Guide\", \"icon_url\": \"spruce:///images/icon_rx_guide\", \"tap_url\": \"spruce:///action/view_treatment_guide?treatment_id=doxy\"} ] } ] }, {\"type\": \"treatment:button_footer\", \"footer_text\": \"If you have any questions about your treatment plan, you can always message your care team.\", \"center_footer_text\": true } ], \"instruction_views\": [{\"type\": \"treatment:card_view\", \"views\": [{\"type\": \"treatment:text\", \"text\": \"Your dermatologist will explain how to use your treatments together in a personalized skin care routine.\", \"style\": \"title1_medium\"}, {\"type\": \"treatment:text\", \"text\": \"They may also share tips and advice on what to expect with your treatment plan.\"} ] }, {\"type\": \"treatment:card_view\", \"views\": [{\"type\": \"treatment:card_title_view\", \"title\": \"Morning Regimen\"}, {\"type\": \"treatment:list_element\", \"element_style\": \"bulleted\", \"number\": 1, \"text\": \"Wash your face with a gentle cleanser.\"}, {\"type\": \"treatment:list_element\", \"element_style\": \"bulleted\", \"number\": 2, \"text\": \"Apply a lightweight moisturizer with SPF 50.\"}, {\"type\": \"treatment:list_element\", \"element_style\": \"bulleted\", \"number\": 3, \"text\": \"Take doxycycline 100mg with breakfast.\"} ] }, {\"type\": \"treatment:card_view\", \"views\": [{\"type\": \"treatment:card_title_view\", \"title\": \"Nighttime Regimen\"}, {\"type\": \"treatment:list_element\", \"element_style\": \"bulleted\", \"number\": 1, \"text\": \"Take doxycycline 100mg with dinner.\"}, {\"type\": \"treatment:list_element\", \"element_style\": \"bulleted\", \"number\": 2, \"text\": \"Wash your face with a gentle cleanser.\"}, {\"type\": \"treatment:list_element\", \"element_style\": \"bulleted\", \"number\": 3, \"text\": \"Dry your face completely.\"}, {\"type\": \"treatment:list_element\", \"element_style\": \"bulleted\", \"number\": 4, \"text\": \"Apply a pea-sized amount of tretinoin cream to entire face.\"}, {\"type\": \"treatment:list_element\", \"element_style\": \"bulleted\", \"number\": 5, \"text\": \"Apply a nighttime moisturizer as needed.\"} ] }, {\"type\": \"treatment:card_view\", \"views\": [{\"type\": \"treatment:card_title_view\", \"title\": \"Resources\"}, {\"type\": \"treatment:large_icon_text_button\", \"text\": \"Daytime Moisturizer\", \"icon_url\": \"http://f.cl.ly/items/093K3B2t3V1l2T472t3J/moistureizer_thumb.png\", \"icon_width\": 60, \"icon_height\": 60, \"tap_url\": \"spruce:///action/view_resource_library_guide?guide_id=96\"}, {\"type\": \"treatment:small_divider\"}, {\"type\": \"treatment:large_icon_text_button\", \"text\": \"Adult Acne\", \"icon_url\": \"http://f.cl.ly/items/172C1Y1b2a2v3G1l3W09/adult_acne_thumb.png\", \"icon_width\": 60, \"icon_height\": 60, \"tap_url\": \"spruce:///action/view_resource_library_guide?guide_id=100\"} ] }, {\"type\": \"treatment:button_footer\", \"footer_text\": \"If you have any questions about your treatment plan, you can always message your care team.\", \"center_footer_text\": true } ] }');
/*!40000 ALTER TABLE `clinical_pathway` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `clinical_pathway_menu`
--

DROP TABLE IF EXISTS `clinical_pathway_menu`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `clinical_pathway_menu` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `json` blob NOT NULL,
  `status` varchar(32) NOT NULL,
  `created` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `status` (`status`,`created`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `clinical_pathway_menu`
--

LOCK TABLES `clinical_pathway_menu` WRITE;
/*!40000 ALTER TABLE `clinical_pathway_menu` DISABLE KEYS */;
INSERT INTO `clinical_pathway_menu` VALUES (1,'{\"title\": \"What are you here to see the doctor for today?\", \"items\": [{\"title\": \"Acne\", \"type\": \"pathway\", \"pathway\": {\"tag\": \"health_condition_acne\"}}]}','ACTIVE','2015-01-21 18:33:09');
/*!40000 ALTER TABLE `clinical_pathway_menu` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2015-05-08 14:01:45
