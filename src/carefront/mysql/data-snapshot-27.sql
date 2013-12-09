-- MySQL dump 10.13  Distrib 5.6.13, for osx10.8 (x86_64)
--
-- Host: dev-db-3.ccvrwjdx3gvp.us-east-1.rds.amazonaws.com    Database: database_32751
-- ------------------------------------------------------
-- Server version	5.6.13-log

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
) ENGINE=InnoDB AUTO_INCREMENT=174 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_text`
--

LOCK TABLES `app_text` WRITE;
/*!40000 ALTER TABLE `app_text` DISABLE KEYS */;
INSERT INTO `app_text` VALUES (1,'reason for visit with doctor','txt_visit_reason'),(2,'acne is the reason for visit','txt_acne_visit_reason'),(3,'something else is reason for visit','txt_something_else_visit_reason'),(4,'hint for typing a symptom or condition','txt_hint_type_symptom'),(5,'duration of acne','txt_acne_length'),(6,'0-6 months for acne length','txt_less_six_months'),(7,'6-12 months for acne length','txt_six_months_one_year_acne_length'),(8,'1-2 years for acne length','txt_one_two_year_acne_length'),(9,'2+ years for acne length','txt_two_plus_year_acne_length'),(10,'is your acne getting worse','txt_acne_worse'),(11,'acne is getting worse response','txt_yes'),(12,'acne is not getting worse response','txt_no'),(13,'helper text to describe what is making acne worse','txt_describe_changes_acne_worse'),(14,'hint text giving examples for what makes acne worse','txt_examples_changes_acne_worse'),(15,'select type of treatments tried for acne','txt_acne_treatments'),(16,'over the counter treatment for acne','txt_otc_acne_treatment'),(17,'prescription treatment for acne','txt_prescription_treatment'),(18,'no treatment tried for acne','txt_no_treatment_acne'),(19,'list medications tried for acne','txt_list_medications_acne'),(20,'type to add treatment','txt_type_add_treatment'),(21,'share anything else w.r.t acne','txt_anything_else_acne'),(22,'hint for anything else you\'d like to tell the doctor','txt_hint_anything_else_acne_treatment'),(23,'question for females to learn about family planning','txt_pregnancy_planning'),(26,'Are you allergic to any medications?','txt_allergic_to_medications'),(29,'hint to add a medication','txt_type_add_medication'),(30,'Your Skin History','txt_skin_history'),(31,'Your Medical History','txt_medical_history'),(32,'question to list medications','txt_list_medications'),(33,'hint to list medications','txt_hint_list_medications'),(34,'question to get social history','txt_get_social_history'),(35,'smoke tobacco','txt_smoke_social_history'),(36,'drink alocohol','txt_alcohol_social_history'),(37,'use tanning beds','txt_tanning_social_history'),(38,'question to learn whether patient has been diagnosed in the past','txt_diagnosed_skin_past'),(39,'listing past skin diagnosis for paitent to chose from','txt_alopecia_diagnosis'),(40,'listing past sking diagnoses for patient to chose from','txt_acne_diagnosis'),(41,'listing past sking diagnoses for patient to chose from','txt_eczema'),(42,'listing past sking diagnoses for patient to chose from','txt_psoriasis_diagnosis'),(43,'listing past sking diagnoses for patient to chose from','txt_rosacea_diagnosis'),(44,'listing past sking diagnoses for patient to chose from','txt_skin_cancer_diagnosis'),(45,'listing past sking diagnoses for patient to chose from','txt_other_diagnosis'),(46,'question to list any medical conditions that patient has been treated for','txt_list_medical_condition'),(47,'hint to prompt user to add a condition','txt_hint_add_condition'),(48,'medical condition list to chose from','txt_arthritis_condition'),(49,'medical condition list to chose from','txt_artificial_heart_valve_condition'),(50,'medical condition list to chose from','txt_artificial_joint_condition'),(51,'medical condition list to chose from','txt_asthma_condition'),(52,'medical condition list to chose from','txt_blood_clots_condition'),(53,'medical condition list to chose from','txt_diabetes_condition'),(54,'medical condition list to chose from','txt_epilepsy_condition'),(55,'medical condition list to chose from','txt_high_bp_condition'),(56,'medical condition list to chose from','txt_high_cholestrol_condition'),(57,'medical condition list to chose from','txt_hiv_condition'),(58,'medical condition list to chose from','txt_heart_attack_condition'),(59,'medical condition list to chose from','txt_heart_murmur_condition'),(60,'medical condition list to chose from','txt_irregular_heartbeat_condition'),(61,'medical condition list to chose from','txt_kidney_disease_condition'),(62,'medical condition list to chose from','txt_liver_disease_condition'),(63,'medical condition list to chose from','txt_lung_disease_condition'),(64,'medical condition list to chose from','txt_lupus_disease_condition'),(65,'medical condition list to chose from','txt_organ_transplant_disease_condition'),(66,'medical condition list to chose from','txt_pacemaker_disease_condition'),(67,'medical condition list to chose from','txt_thyroid_problems_condition'),(68,'medical condition list to chose from','txt_other_condition_condition'),(69,'medical condition list to chose from','txt_no_condition'),(70,'question to determine where the patient is experiencing acne','txt_acne_location'),(71,'face location for acne','txt_face_acne_location'),(72,'chest location for acne','txt_chest_acne_location'),(73,'back location for acne','txt_back_acne_location'),(74,'other locations for acne','txt_other_acne_location'),(75,'title for face section of photo tips','txt_face_photo_tips_title'),(76,'description for face section of photo taking','txt_photo_tips_description'),(77,'tip to remove glasses','txt_remove_glasses_tip'),(78,'tip to pull hair back','txt_pull_hair_back_tip'),(79,'tip to have no makeup','txt_no_makeup_tip'),(80,'title for chest section photo tips','txt_chest_photo_tips_title'),(81,'tip to remove jewellery','txt_remove_jewellery_photo_tip'),(82,'face front label','txt_face_front'),(83,'profile left label','txt_profile_left'),(84,'profile right label','txt_profile_right'),(85,'chest label','txt_chest'),(86,'back lebel','txt_back'),(87,'title for photo section','txt_photo_section_title'),(88,'short description of reason for visit','txt_short_reason_visit'),(89,'short description for length of time patient has been experiencing acne','txt_short_acne_length'),(90,'short description of other symptoms that the patient is attempting to use the app for ','txt_short_other_symptoms'),(91,'short description of whether or not acne is getting worse','txt_short_acne_worse'),(92,'short description of changes that would be making acne worse','txt_short_changes_acne_worse'),(93,'short description of previous types of treatments tried','txt_short_prev_type_treatment'),(94,'short description of previous list of treatments that have been tried','txt_short_prev_list_treatment'),(95,'short description of anything else patient would like to tell doctor about cane','txt_short_anything_else_acne'),(96,'short description of all the places that the patient marked acne is being present on','txt_short_photo_locations'),(97,'short description of whether patient is planning pregnancy','txt_short_pregnant'),(98,'short description of whether patient is alergic to medications','txt_allergic_medications'),(99,'short description to list any medications patient is currently taking','txt_short_list_medications'),(100,'short description to describe social history of patient','txt_short_social_history'),(101,'short description for previous skin diagnosis','txt_short_prev_skin_diagnosis'),(102,'short description for patient to describe medical conditions that they have been treated for','txt_short_medical_condition'),(103,'prompt to take photo of treatment','txt_take_photo_treatment'),(104,'short description for a list of medications that patient is allergic to','txt_short_allergic_medications_list'),(105,'short description for front face photo of patient','txt_short_face_photo'),(106,'short description for chest photos of patient','txt_short_chest_photo'),(107,'short description for back photo of patient','txt_short_back_photo'),(108,'short description for other photo of patient','txt_short_other_photo'),(109,'other lable for photo taking','txt_other'),(110,'how effective was this treatment','txt_effective_treatment'),(111,'answer option','txt_not_very'),(112,'answer option','txt_somewhat'),(113,'answer option','txt_very'),(114,'are you currently using this treatment','txt_current_treatment'),(115,'less than 1 month','txt_one_or_less'),(116,'2-5 months','txt_two_five_months'),(117,'6-11 months','txt_six_eleven_months'),(118,'12+ months','txt_twelve_plus_months'),(119,'not very effective','txt_not_very_effective'),(120,'somewhat effective','txt_somewhat_effective'),(121,'very effective','txt_very_effective'),(122,'currently using it','txt_current_using'),(123,'not currently using it','txt_not_currently_using'),(124,'Used for less than 1 month','txt_used_less_1_month'),(125,'Used for 2-5 months','txt_used_two_five_months'),(126,'Used for 6-11 months','txt_used_six_eleven_months'),(127,'Used for over a year','txt_used_twelve_plus_months'),(128,'question for length of treatment','txt_treatment_length'),(129,'txt for when you first started experiencing acne','txt_first_acne_experience'),(130,'txt response of during puberty','txt_during_puberty'),(131,'txt response of within last six months','txt_within_last_six_months'),(132,'txt response of 1-2 years ago','txt_one_two_years_ago'),(133,'txt response of more than 2 years ago','txt_more_than_two_years'),(134,'txt summary for onset of symptoms','txt_onset_symptoms'),(135,'txt for asking the user if they are experiencing acne symptoms','txt_acne_symtpoms'),(136,'txt for response of acne being painful to touch','txt_painful_touch'),(137,'txt for response of acne being scarring','txt_scarring'),(138,'txt for response of acne causing discoloration','txt_discoloration'),(139,'txt for summarizing additional symptoms','txt_additional_symptoms'),(140,'txt for asking female patients if their acne gets worse with periods','txt_acne_worse_period'),(141,'txt for asking female patients if their periods are regular','txt_periods_regular'),(142,'txt for summarizing information about txt_menstrual_cycle','txt_menstrual_cycle'),(143,'txt for question to descibe skin','txt_skin_description'),(144,'txt for response to skin description as normal','txt_normal_skin'),(145,'txt for response to skin description as oily','txt_oily_skin'),(146,'txt for response to skin description as dry','txt_dry_skin'),(147,'txt for response to skin description as combination','txt_combination_skin'),(148,'txt for summarizing skin type','txt_skin_type'),(149,'txt for determining whether patient has been allergic to topical medication','txt_allergy_topical_medication'),(150,'txt summary for determining whether patient has been allergic to topical medication','txt_summary_allergy_topical_medication'),(151,'txt for determining any other conditions patient may have been diagnosed for in the past','txt_other_condition_acne'),(152,'txt for determining any other conditions patient may have been diagnosed for in the past','txt_summary_other_condition_acne'),(153,'txt response for determining any other conditions patient may have been diagnosed for in the past','txt_gasitris'),(154,'txt response for determining any other conditions patient may have been diagnosed for in the past','txt_colitis'),(155,'txt response for determining any other conditions patient may have been diagnosed for in the past','txt_kidney_disease'),(156,'txt response for determining any other conditions patient may have been diagnosed for in the past','txt_lupus'),(157,'txt summary for treatment not effective','txt_answer_summary_not_effective'),(158,'txt summary for treatment somewhat effective','txt_answer_summary_somewhat_effective'),(159,'txt summary for treatment very effective','txt_answer_summary_very_effective'),(160,'txt summary for not currently using treatment','txt_answer_summary_not_using'),(161,'txt summary for using current treatment','txt_answer_summary_using'),(162,'txt summary for using treatment less than a month','txt_answer_summary_less_month'),(163,'txt summary for using treatment 2-5 months','txt_answer_summary_two_five_months'),(164,'txt summary for using treamtent 6-11 months','txt_answer_summary_six_eleven_months'),(165,'txt summary for using treatment 12+ months','txt_answer_summary_twelve_plus_months'),(166,'txt for prompting user to add treatment','txt_add_treatment'),(167,'txt for prompting user to add medication','txt_add_medication'),(168,'txt for prompting user to take a photo of the medication','txt_take_photo_medication'),(169,'txt for button when adding medication','txt_add_button_medication'),(170,'txt for button when adding treatment','txt_add_button_treatment'),(171,'txt for saving changes when adding medication or treatment','txt_save_changes'),(172,'txt for button to remove treatment','txt_remove_treatment'),(173,'txt for button to remove medication','txt_remove_medication');
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
) ENGINE=InnoDB AUTO_INCREMENT=180 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `localized_text`
--

LOCK TABLES `localized_text` WRITE;
/*!40000 ALTER TABLE `localized_text` DISABLE KEYS */;
INSERT INTO `localized_text` VALUES (4,1,'What\'s the reason for your visit with Dr. Who today?',1),(5,1,'Something else',3),(6,1,'Acne',2),(7,1,'Type a symptom or condition',4),(8,1,'How long have you been experiencing acne symptoms?',5),(9,1,'0-6 months',6),(10,1,'6-123 months',7),(11,1,'1-2 years',8),(12,1,'2+ years',9),(13,1,'Is your acne getting worse?',10),(14,1,'Yes',11),(15,1,'No',12),(16,1,'Describe any recent changes that could be making your acne worse:',13),(18,1,'Examples: recreational activities, new cosmetics or toiletries, warmer weather, stress',14),(19,1,'Select what type of treatments you\'ve previously tried for your acne:',15),(20,1,'Over the counter',16),(21,1,'Prescription',17),(22,1,'No treatments tried',18),(23,1,'List any prescription or over the counter treatments that you\'ve tried for acne',19),(24,1,'Type to add a treatment',20),(25,1,'Is there anything else you\'d like to share about your acne with Dr. wHO?',21),(26,1,'This question is optional but feel free to share anything else about your acne that you think the doctor should know..',22),(27,1,'Are you pregnant, planning a pregnancy or nursing?',23),(28,1,'Are you allergic to any medications?',26),(29,1,'Type to add a medication',29),(30,1,'Your Skin History',30),(31,1,'Your Medical History',31),(32,1,'List any medications you are currently taking:',32),(33,1,'Include birth control, over the counter medications, vitamins or herbal supplements that you may be currently taking.',33),(34,1,'Select which if any of the following activities you do regularly:',34),(35,1,'Smoke tobacco',35),(36,1,'Drink alcohol',36),(37,1,'Use tanning beds or sunbath',37),(38,1,'Have you been diagnosed for a skin condition in the past?',38),(39,1,'Alopecia',39),(40,1,'Acne',40),(42,1,'Eczema',41),(43,1,'Psoriasis',42),(44,1,'Rosacea',43),(45,1,'Skin Cancer',44),(47,1,'Other',45),(48,1,'List any medical condition that you currently have or have been treated for:',46),(50,1,'Type to add a condition',47),(51,1,'Arthritis',48),(53,1,'Artifical Heart Valve',49),(55,1,'Artifical Joint',50),(56,1,'Asthma',51),(57,1,'Blood Clots',52),(58,1,'Diabetes',53),(59,1,'Epilepsy or Seizures',54),(60,1,'High Blood Pressure',55),(61,1,'High Cholestrol',56),(62,1,'HIV/AIDs',57),(63,1,'Heart Attack',58),(64,1,'Heart Murmur',59),(66,1,'Irregular Heartbeat',60),(67,1,'Kidney Disease',61),(68,1,'Liver Disease',62),(69,1,'Lung Disease',63),(70,1,'Lupus',64),(71,1,'Organ Transplant',65),(72,1,'Pacemaker',66),(73,1,'Thyroid Problems',67),(74,1,'Other Condition Not Listed',68),(75,1,'No past or present conditions',69),(76,1,'Photos for Diagnosis',87),(77,1,'We need to know where you\'re experiencing acne so we can take the right photos.',70),(78,1,'Face',71),(79,1,'Chest',72),(80,1,'Back',73),(81,1,'Other',74),(82,1,'Up First: Face Photos',75),(83,1,'Remember these photos are for diagnosis purposes. The clearer your photo the easier it is for the doctor to make a diagnosis.',76),(84,1,'Remove glasses or hats',77),(85,1,'Pull back any hair covering your face',78),(86,1,'No make up',79),(87,1,'Remve any jewellery or clothing that may be covering your chest (except under garments)',81),(88,1,'Next: Chest Photos',80),(89,1,'Reason for visit',88),(90,1,'Length of time with acne symptoms',89),(91,1,'Other symptoms or conditions patient wants diagnosed',90),(92,1,'Worsening symptoms',91),(93,1,'Recent changes making acne worse',92),(94,1,'Type of treatments',93),(95,1,'OTC and Prescriptions tried',94),(96,1,'Additional info patient shared',95),(97,1,'Location of symptoms',96),(98,1,'Pregnant/Nursing',97),(99,1,'Medication Allergies',98),(100,1,'Current medications',99),(101,1,'Social History',100),(102,1,'Skin Conditions',101),(103,1,'Other Conditions',102),(104,1,'Or take a photo of the treatment',103),(105,1,'Face photos of patient',105),(106,1,'Chest photos of patient',106),(107,1,'Back photos of patient',107),(108,1,'Other photos of patient',108),(109,1,'Other',109),(110,1,'Front',82),(111,1,'Profile Left',83),(112,1,'Profile Right',84),(113,1,'Chest',85),(114,1,'How effective was this treatment?',110),(115,1,'Not Very',111),(116,1,'Somewhat',112),(117,1,'Very',113),(118,1,'Are you currently using this treatment?',114),(119,1,'1 or less',115),(120,1,'2-5',116),(121,1,'6-11',117),(122,1,'12+',118),(123,1,'Not very effective',119),(124,1,'Somewhat effective',120),(125,1,'Very effective',121),(126,1,'Currently using it',122),(127,1,'Not currently using it',123),(128,1,'Used for less than 1 month',124),(129,1,'Used for 2-5 months',125),(131,1,'Used for 6-11 months',126),(132,1,'Used for over a year',127),(133,1,'Approximately how many months did you use this treatment for?',128),(134,1,'When did you first begin experiencing acne?',129),(135,1,'During puberty',130),(136,1,'Within the last six months',131),(137,1,'1-2 years ago',132),(138,1,'More than 2 years ago',133),(139,1,'Onset of symptoms',134),(140,1,'Are you experiencing any of the following symptoms with your acne?',135),(141,1,'Painful to the touch',136),(142,1,'Scarring',137),(143,1,'Discoloration',138),(144,1,'Additional Symptoms',139),(145,1,'Does your acne get worse with your period?',140),(146,1,'Are your periods regular?',141),(147,1,'Menstrual cycle',142),(148,1,'How would you describe your skin?',143),(149,1,'Normal',144),(150,1,'Oily',145),(151,1,'Dry',146),(152,1,'Combination',147),(153,1,'Skin type',148),(154,1,'Have you ever had an allergic reaction to a topical medication?',149),(155,1,'Topical Medication Allergies',150),(156,1,'Do you currently have or have been treated for any of the following conditions?',151),(157,1,'Other conditions',152),(158,1,'Gasitris',153),(159,1,'Colitis',154),(160,1,'Kidney Disease',155),(161,1,'Lupus',156),(162,1,'Add a medication',104),(163,1,'Not very effective',157),(164,1,'Somewhat effective',158),(165,1,'Very effective',159),(166,1,'Not currently using it',160),(167,1,'Currently using it',161),(168,1,'Used for less than one month',162),(169,1,'Used for 2-5 months',163),(170,1,'Used for 6-11 months',164),(171,1,'Used for 12+ months',165),(172,1,'Add a Treatment',166),(173,1,'Add a Medication',167),(174,1,'Or take a photo of the medication',168),(175,1,'Add Medication',169),(176,1,'Add Treatment',170),(177,1,'Save Changes',171),(178,1,'Remove Treatment',172),(179,1,'Remove Medication',173);
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
  PRIMARY KEY (`id`),
  UNIQUE KEY `otype` (`atype`)
) ENGINE=InnoDB AUTO_INCREMENT=15 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `answer_type`
--

LOCK TABLES `answer_type` WRITE;
/*!40000 ALTER TABLE `answer_type` DISABLE KEYS */;
INSERT INTO `answer_type` VALUES (5,'a_type_autocomplete_entry'),(4,'a_type_dropdown_entry'),(2,'a_type_free_text'),(1,'a_type_multiple_choice'),(11,'a_type_photo_entry_back'),(12,'a_type_photo_entry_chest'),(8,'a_type_photo_entry_face_left'),(7,'a_type_photo_entry_face_middle'),(10,'a_type_photo_entry_face_right'),(13,'a_type_photo_entry_other'),(6,'a_type_photo_to_text_entry'),(14,'a_type_segmented_control'),(3,'a_type_single_entry');
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
-- Table structure for table `health_condition`
--

DROP TABLE IF EXISTS `health_condition`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `health_condition` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `comment` varchar(600) NOT NULL,
  `health_condition_tag` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `treatment_tag` (`health_condition_tag`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `health_condition`
--

LOCK TABLES `health_condition` WRITE;
/*!40000 ALTER TABLE `health_condition` DISABLE KEYS */;
INSERT INTO `health_condition` VALUES (1,'health_condition_acne','health_condition_acne');
/*!40000 ALTER TABLE `health_condition` ENABLE KEYS */;
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
  `health_condition_id` int(10) unsigned DEFAULT NULL,
  `section_tag` varchar(250) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `section_tag` (`section_tag`),
  KEY `section_title_app_text_id` (`section_title_app_text_id`),
  KEY `health_condition_id` (`health_condition_id`),
  CONSTRAINT `section_ibfk_2` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`),
  CONSTRAINT `section_ibfk_1` FOREIGN KEY (`section_title_app_text_id`) REFERENCES `app_text` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `section`
--

LOCK TABLES `section` WRITE;
/*!40000 ALTER TABLE `section` DISABLE KEYS */;
INSERT INTO `section` VALUES (1,30,'skin history section',1,'section_skin_history'),(2,31,'medical history section',NULL,'section_medical_history'),(3,87,'photos for diagnosis',1,'section_photo_diagnosis');
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
  PRIMARY KEY (`id`),
  UNIQUE KEY `qtype` (`qtype`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `question_type`
--

LOCK TABLES `question_type` WRITE;
/*!40000 ALTER TABLE `question_type` DISABLE KEYS */;
INSERT INTO `question_type` VALUES (9,'q_type_autocomplete'),(3,'q_type_compound'),(2,'q_type_free_text'),(1,'q_type_multiple_choice'),(7,'q_type_multiple_photo'),(8,'q_type_segmented_control'),(5,'q_type_single_entry'),(4,'q_type_single_photo'),(6,'q_type_single_select');
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
  `qtype_id` int(10) unsigned NOT NULL,
  `qtext_app_text_id` int(10) unsigned DEFAULT NULL,
  `qtext_short_text_id` int(10) unsigned DEFAULT NULL,
  `subtext_app_text_id` int(10) unsigned DEFAULT NULL,
  `question_tag` varchar(250) NOT NULL,
  `parent_question_id` int(10) unsigned DEFAULT NULL,
  `required` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `question_tag` (`question_tag`),
  KEY `qtype_id` (`qtype_id`),
  KEY `subtext_app_text_id` (`subtext_app_text_id`),
  KEY `qtext_app_text_id` (`qtext_app_text_id`),
  KEY `qtext_short_text_id` (`qtext_short_text_id`),
  KEY `parent_question_id` (`parent_question_id`),
  CONSTRAINT `question_ibfk_1` FOREIGN KEY (`qtype_id`) REFERENCES `question_type` (`id`),
  CONSTRAINT `question_ibfk_2` FOREIGN KEY (`subtext_app_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `question_ibfk_3` FOREIGN KEY (`qtext_app_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `question_ibfk_4` FOREIGN KEY (`qtext_short_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `question_ibfk_5` FOREIGN KEY (`parent_question_id`) REFERENCES `question` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=35 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `question`
--

LOCK TABLES `question` WRITE;
/*!40000 ALTER TABLE `question` DISABLE KEYS */;
INSERT INTO `question` VALUES (1,6,1,88,NULL,'q_reason_visit',NULL,NULL),(2,5,NULL,90,NULL,'q_condition_for_diagnosis',NULL,NULL),(3,6,5,89,NULL,'q_acne_length',NULL,NULL),(4,6,10,91,NULL,'q_acne_worse',NULL,NULL),(6,2,13,92,NULL,'q_changes_acne_worse',NULL,NULL),(7,1,15,93,NULL,'q_acne_prev_treatment_types',NULL,NULL),(8,9,19,94,NULL,'q_acne_prev_treatment_list',NULL,NULL),(9,2,21,95,NULL,'q_anything_else_acne',NULL,NULL),(10,6,23,97,NULL,'q_pregnancy_planning',NULL,NULL),(11,1,26,98,NULL,'q_allergic_medications',NULL,NULL),(12,9,29,104,NULL,'q_allergic_medication_entry',NULL,NULL),(13,9,32,99,33,'q_current_medications_entry',NULL,NULL),(14,1,34,100,NULL,'q_social_history',NULL,NULL),(15,6,38,101,NULL,'q_prev_skin_condition_diagnosis',NULL,NULL),(16,3,46,102,NULL,'q_prev_med_condition_diagnosis',NULL,NULL),(17,1,NULL,101,NULL,'q_list_prev_skin_condition_diagnosis',NULL,NULL),(18,1,70,96,NULL,'q_acne_location',NULL,NULL),(19,4,NULL,105,NULL,'q_face_photo_intake',NULL,NULL),(20,4,NULL,106,NULL,'q_chest_photo_intake',NULL,NULL),(21,4,NULL,107,NULL,'q_back_photo_intake',NULL,NULL),(22,7,NULL,108,NULL,'q_other_photo_intake',NULL,NULL),(24,8,110,NULL,NULL,'q_effective_treatment',8,NULL),(25,8,114,NULL,NULL,'q_using_treatment',8,NULL),(26,8,128,NULL,NULL,'q_length_treatment',8,NULL),(27,6,129,134,NULL,'q_onset_acne',NULL,1),(28,1,135,139,NULL,'q_acne_symptoms',NULL,1),(29,6,140,142,NULL,'q_acne_worse_period',NULL,0),(30,6,141,NULL,NULL,'q_periods_regular',29,0),(31,6,143,148,NULL,'q_skin_description',NULL,1),(32,6,149,150,NULL,'q_topical_allergic_medications',NULL,1),(33,6,151,152,NULL,'q_other_conditions_acne',NULL,1),(34,9,29,150,NULL,'q_topical_allergies_medication_entry',NULL,0);
/*!40000 ALTER TABLE `question` ENABLE KEYS */;
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
  `atype_id` int(10) unsigned NOT NULL,
  `potential_answer_tag` varchar(250) NOT NULL,
  `ordering` int(10) unsigned NOT NULL,
  `answer_summary_text_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `potential_outcome_tag` (`potential_answer_tag`),
  UNIQUE KEY `question_id_2` (`question_id`,`ordering`),
  KEY `otype_id` (`atype_id`),
  KEY `outcome_localized_text` (`answer_localized_text_id`),
  KEY `answer_summary_text_id` (`answer_summary_text_id`),
  CONSTRAINT `potential_answer_ibfk_3` FOREIGN KEY (`answer_summary_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `potential_answer_ibfk_1` FOREIGN KEY (`atype_id`) REFERENCES `answer_type` (`id`),
  CONSTRAINT `potential_answer_ibfk_2` FOREIGN KEY (`question_id`) REFERENCES `question` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=102 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `potential_answer`
--

LOCK TABLES `potential_answer` WRITE;
/*!40000 ALTER TABLE `potential_answer` DISABLE KEYS */;
INSERT INTO `potential_answer` VALUES (1,1,2,1,'a_acne',0,NULL),(2,1,3,1,'a_something_else',1,NULL),(3,2,NULL,3,'a_condition_entry',0,NULL),(4,3,6,1,'a_less_six_months',0,NULL),(5,3,7,1,'a_six_twelve_months',1,NULL),(6,3,8,1,'a_one_twa_years',2,NULL),(7,3,9,1,'a_twa_plus_years',3,NULL),(8,4,11,1,'a_yes_acne_worse',0,NULL),(9,4,12,1,'a_na_acne_worse',1,NULL),(12,7,17,1,'a_prescription_prev_treatment_type',0,NULL),(13,7,16,1,'a_otc_prev_treatment_type',1,NULL),(14,7,18,1,'a_na_prev_treatment_type',2,NULL),(18,10,11,1,'a_yes_pregnancy_planning',0,NULL),(19,10,12,1,'a_na_pregnancy_planning',1,NULL),(20,11,11,1,'a_yes_allergic_medications',0,NULL),(21,11,12,1,'a_na_allergic_medications',1,NULL),(24,14,35,1,'a_smoke_social_history',0,NULL),(25,14,36,1,'a_alcohol_social_history',1,NULL),(26,14,37,1,'a_tanning_social_history',2,NULL),(27,15,11,1,'a_yes_prev_skin_diagnosis',0,NULL),(28,15,12,1,'a_na_prev_skin_diagnosis',1,NULL),(29,17,39,1,'a_alopecia_skin_diagnosis',0,NULL),(30,17,40,1,'a_acne_skin_diagnosis',1,NULL),(31,17,41,1,'a_eczema_skin_diagnosis',2,NULL),(32,17,42,1,'a_psoriasis_skin_diagnosis',3,NULL),(33,17,43,1,'a_rosacea_skin_diagnosis',4,NULL),(34,17,44,1,'a_skin_cancer_diagnosis',5,NULL),(35,17,45,1,'a_other_skin_iagnosis',6,NULL),(36,16,48,1,'a_arthritis_diagnosis',0,NULL),(37,16,49,1,'a_heart_valve_diagnosis',1,NULL),(38,16,50,1,'a_artificial_join__diagnosis',2,NULL),(39,16,51,1,'a_asthma_diagnosis',3,NULL),(40,16,52,1,'a_blood_clots_diagnosis',4,NULL),(41,16,53,1,'a_diabetes_diagnosis',5,NULL),(42,16,54,1,'a_epilepsey_diagnosis',6,NULL),(43,16,55,1,'a_high_blood_pressure_diagnosis',7,NULL),(44,16,56,1,'a_high_cholestrol_diagnosis',8,NULL),(45,16,57,1,'a_hiv_diagnosis',9,NULL),(46,16,58,1,'a_heart_attack_diagnosis',10,NULL),(47,16,59,1,'a_heart_murmur_diagnosis',11,NULL),(48,16,60,1,'a_irregular_heart_beat_skin_diagnosis',12,NULL),(49,16,61,1,'a_kidney_disease_diagnosis',13,NULL),(50,16,62,1,'a_liver_disease_diagnosis',14,NULL),(51,16,63,1,'a_lung_disease_diagnosis',15,NULL),(52,16,64,1,'a_lupus_disease_diagnosis',16,NULL),(53,16,65,1,'a_organ_transplant_diagnosis',17,NULL),(55,16,66,1,'a_pacemaker_diagnosis',18,NULL),(56,16,67,1,'a_thyroid_diagnosis',19,NULL),(57,16,68,1,'a_other_skin_diagnosis',20,NULL),(58,16,69,1,'a_none_skin_diagnosis',21,NULL),(59,18,71,1,'a_face_acne_location',0,NULL),(60,18,72,1,'a_chest_acne_location',1,NULL),(61,18,73,1,'a_back_acne_location',2,NULL),(62,18,74,1,'a_other_acne_location',3,NULL),(63,19,82,7,'a_face_front_phota_intake',0,NULL),(64,19,84,10,'a_face_right_phota_intake',1,NULL),(65,19,83,8,'a_face_left_phota_intake',2,NULL),(66,20,85,12,'a_chest_phota_intake',0,NULL),(68,21,86,11,'a_back_phota_intake',0,NULL),(69,22,109,13,'a_other_phota_intake',0,NULL),(70,24,111,14,'a_effective_treatment_not_very',0,157),(71,24,112,14,'a_effective_treatment_somewhat',1,158),(72,24,113,14,'a_effective_treatment_very',2,159),(73,25,11,14,'a_using_treatment_yes',0,161),(75,25,12,14,'a_using_treatment_no',1,160),(76,26,115,14,'a_length_treatment_less_one',0,162),(77,26,116,14,'a_length_treatment_two_five_months',1,163),(78,26,117,14,'a_length_treatment_six_eleven_months',2,164),(79,26,118,14,'a_length_treatment_twelve_plus_months',3,165),(80,27,130,1,'a_puberty',0,NULL),(81,27,131,1,'a_onset_six_months',1,NULL),(82,27,132,1,'a_onset_one_two_years',2,NULL),(83,27,133,1,'a_onset_more_two_years',3,NULL),(84,28,136,1,'a_painful_touch',0,NULL),(85,28,137,1,'a_scarring',1,NULL),(86,28,138,1,'a_discoloration',2,NULL),(87,29,11,1,'a_acne_worse_yes',0,NULL),(88,29,12,1,'a_acne_worse_no',1,NULL),(89,30,11,1,'a_periods_regular_yes',0,NULL),(90,30,12,1,'a_periods_regular_no',1,NULL),(91,31,144,1,'a_normal_skin',0,NULL),(92,31,145,1,'a_oil_skin',1,NULL),(93,31,146,1,'a_dry_skin',2,NULL),(94,31,147,1,'a_combination_skin',3,NULL),(95,32,11,1,'a_topical_allergic_medication_yes',0,NULL),(96,32,12,1,'a_topical_allergic_medication_no',1,NULL),(97,33,153,1,'a_other_condition_acne_gastiris',0,NULL),(98,33,154,1,'a_other_condition_acne_colitis',1,NULL),(99,33,155,1,'a_other_condition_acne_kidney_condition',2,NULL),(100,33,156,1,'a_other_condition_acne_lupus',3,NULL);
/*!40000 ALTER TABLE `potential_answer` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `photo_tips`
--

DROP TABLE IF EXISTS `photo_tips`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `photo_tips` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `photo_tips_tag` varchar(100) NOT NULL,
  `photo_url_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `photo_tips_tag` (`photo_tips_tag`),
  KEY `photo_url_id` (`photo_url_id`),
  CONSTRAINT `photo_tips_ibfk_1` FOREIGN KEY (`photo_url_id`) REFERENCES `object_storage` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `photo_tips`
--

LOCK TABLES `photo_tips` WRITE;
/*!40000 ALTER TABLE `photo_tips` DISABLE KEYS */;
/*!40000 ALTER TABLE `photo_tips` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `patient_layout_version`
--

DROP TABLE IF EXISTS `patient_layout_version`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_layout_version` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `object_storage_id` int(10) unsigned NOT NULL,
  `language_id` int(10) unsigned NOT NULL,
  `layout_version_id` int(10) unsigned NOT NULL,
  `status` varchar(250) NOT NULL,
  `health_condition_id` int(10) unsigned NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified_date` timestamp NOT NULL ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `layout_version_id` (`layout_version_id`),
  KEY `language_id` (`language_id`),
  KEY `object_storage_id` (`object_storage_id`),
  KEY `treatment_id` (`health_condition_id`),
  CONSTRAINT `patient_layout_version_ibfk_1` FOREIGN KEY (`layout_version_id`) REFERENCES `layout_version` (`id`),
  CONSTRAINT `patient_layout_version_ibfk_2` FOREIGN KEY (`language_id`) REFERENCES `languages_supported` (`id`),
  CONSTRAINT `patient_layout_version_ibfk_3` FOREIGN KEY (`object_storage_id`) REFERENCES `object_storage` (`id`),
  CONSTRAINT `patient_layout_version_ibfk_4` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=43 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `patient_layout_version`
--

LOCK TABLES `patient_layout_version` WRITE;
/*!40000 ALTER TABLE `patient_layout_version` DISABLE KEYS */;
INSERT INTO `patient_layout_version` VALUES (8,34,1,15,'DEPCRECATED',1,'2013-11-08 19:13:34','0000-00-00 00:00:00'),(9,36,1,16,'DEPCRECATED',1,'2013-11-08 19:13:34','0000-00-00 00:00:00'),(10,38,1,17,'DEPCRECATED',1,'2013-11-08 19:13:34','0000-00-00 00:00:00'),(11,40,1,18,'DEPCRECATED',1,'2013-11-08 19:13:34','0000-00-00 00:00:00'),(12,42,1,19,'DEPCRECATED',1,'2013-11-08 19:13:34','2013-11-08 19:22:26'),(13,44,1,20,'DEPCRECATED',1,'2013-11-08 19:22:21','2013-11-11 05:47:05'),(14,46,1,21,'DEPCRECATED',1,'2013-11-11 05:47:04','2013-11-11 05:57:41'),(15,48,1,22,'DEPCRECATED',1,'2013-11-11 05:57:40','2013-11-11 05:58:51'),(16,51,1,24,'DEPCRECATED',1,'2013-11-11 05:58:50','2013-11-11 06:02:30'),(17,53,1,25,'DEPCRECATED',1,'2013-11-11 06:02:29','2013-11-11 06:07:04'),(18,56,1,26,'DEPCRECATED',1,'2013-11-11 06:07:03','2013-11-12 15:02:06'),(19,58,1,27,'DEPCRECATED',1,'2013-11-12 15:02:05','2013-11-12 15:34:18'),(20,60,1,28,'DEPCRECATED',1,'2013-11-12 15:34:17','2013-11-12 15:34:49'),(21,63,1,29,'DEPCRECATED',1,'2013-11-12 15:34:48','2013-11-12 15:34:50'),(22,64,1,30,'DEPCRECATED',1,'2013-11-12 15:34:50','2013-11-12 15:38:15'),(23,66,1,31,'DEPCRECATED',1,'2013-11-12 15:38:15','2013-11-12 15:39:13'),(24,68,1,32,'DEPCRECATED',1,'2013-11-12 15:39:12','2013-11-12 17:02:20'),(25,70,1,33,'DEPCRECATED',1,'2013-11-12 17:02:19','2013-11-12 17:04:08'),(26,72,1,34,'DEPCRECATED',1,'2013-11-12 17:04:08','2013-11-12 17:15:20'),(27,74,1,35,'DEPCRECATED',1,'2013-11-12 17:15:19','2013-11-12 19:36:52'),(28,76,1,36,'DEPCRECATED',1,'2013-11-12 19:36:52','2013-11-17 00:30:53'),(29,106,1,37,'DEPCRECATED',1,'2013-11-17 00:30:52','2013-11-17 00:31:22'),(30,108,1,38,'DEPCRECATED',1,'2013-11-17 00:31:21','2013-11-17 00:48:23'),(31,110,1,39,'DEPCRECATED',1,'2013-11-17 00:48:22','2013-11-17 19:25:25'),(32,112,1,40,'DEPCRECATED',1,'2013-11-17 19:25:24','2013-11-17 19:28:23'),(33,114,1,41,'DEPCRECATED',1,'2013-11-17 19:28:22','2013-11-17 19:36:07'),(34,116,1,42,'DEPCRECATED',1,'2013-11-17 19:36:06','2013-11-20 01:30:08'),(35,121,1,44,'DEPCRECATED',1,'2013-11-20 01:30:07','2013-11-20 01:38:20'),(36,123,1,45,'DEPCRECATED',1,'2013-11-20 01:38:10','2013-11-20 21:04:04'),(37,125,1,46,'DEPCRECATED',1,'2013-11-20 21:04:03','2013-11-24 02:02:41'),(38,138,1,52,'DEPCRECATED',1,'2013-11-24 02:02:41','2013-11-24 02:05:20'),(39,140,1,53,'DEPCRECATED',1,'2013-11-24 02:05:19','2013-11-24 02:09:31'),(40,142,1,54,'DEPCRECATED',1,'2013-11-24 02:09:30','2013-11-24 02:11:37'),(41,144,1,55,'DEPCRECATED',1,'2013-11-24 02:11:36','2013-11-24 02:21:03'),(42,146,1,56,'ACTIVE',1,'2013-11-24 02:21:01','2013-11-24 02:21:04');
/*!40000 ALTER TABLE `patient_layout_version` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `object_storage`
--

DROP TABLE IF EXISTS `object_storage`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `object_storage` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `bucket` varchar(100) NOT NULL,
  `storage_key` varchar(100) NOT NULL,
  `status` varchar(100) NOT NULL,
  `region_id` int(10) unsigned NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified_date` timestamp NOT NULL ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `region_id` (`region_id`,`storage_key`,`bucket`,`status`),
  CONSTRAINT `object_storage_ibfk_1` FOREIGN KEY (`region_id`) REFERENCES `region` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=162 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `object_storage`
--

LOCK TABLES `object_storage` WRITE;
/*!40000 ALTER TABLE `object_storage` DISABLE KEYS */;
INSERT INTO `object_storage` VALUES (33,'carefront-layout','1383846854','ACTIVE',1,'2013-11-08 18:59:58','2013-11-08 19:09:51'),(34,'carefront-client-layout','1383846875','ACTIVE',1,'2013-11-08 18:59:58','0000-00-00 00:00:00'),(35,'carefront-layout','1383852476','ACTIVE',1,'2013-11-08 18:59:58','0000-00-00 00:00:00'),(36,'carefront-client-layout','1383852497','ACTIVE',1,'2013-11-08 18:59:58','0000-00-00 00:00:00'),(37,'carefront-layout','1383934286','ACTIVE',1,'2013-11-08 18:59:58','0000-00-00 00:00:00'),(38,'carefront-client-layout','1383934320','ACTIVE',1,'2013-11-08 18:59:58','0000-00-00 00:00:00'),(39,'carefront-layout','1383937634','0',1,'2013-11-08 19:07:15','0000-00-00 00:00:00'),(40,'carefront-client-layout','1383937647','0',1,'2013-11-08 19:07:28','0000-00-00 00:00:00'),(41,'carefront-layout','1383937817','ACTIVE',1,'2013-11-08 19:10:18','2013-11-08 19:10:19'),(42,'carefront-client-layout','1383937831','ACTIVE',1,'2013-11-08 19:10:33','2013-11-08 19:10:34'),(43,'carefront-layout','1383938460','ACTIVE',1,'2013-11-08 19:21:02','2013-11-08 19:21:05'),(44,'carefront-client-layout','1383938531','ACTIVE',1,'2013-11-08 19:22:13','2013-11-08 19:22:20'),(45,'carefront-layout','1384148805','ACTIVE',1,'2013-11-11 05:46:45','2013-11-11 05:46:47'),(46,'carefront-client-layout','1384148822','ACTIVE',1,'2013-11-11 05:47:02','2013-11-11 05:47:03'),(47,'carefront-layout','1384149443','ACTIVE',1,'2013-11-11 05:57:23','2013-11-11 05:57:25'),(48,'carefront-client-layout','1384149458','ACTIVE',1,'2013-11-11 05:57:38','2013-11-11 05:57:40'),(49,'carefront-layout','1384149498','ACTIVE',1,'2013-11-11 05:58:18','2013-11-11 05:58:19'),(50,'carefront-layout','1384149512','ACTIVE',1,'2013-11-11 05:58:32','2013-11-11 05:58:34'),(51,'carefront-client-layout','1384149529','ACTIVE',1,'2013-11-11 05:58:49','2013-11-11 05:58:50'),(52,'carefront-layout','1384149729','ACTIVE',1,'2013-11-11 06:02:09','2013-11-11 06:02:11'),(53,'carefront-client-layout','1384149748','ACTIVE',1,'2013-11-11 06:02:28','2013-11-11 06:02:29'),(54,'carefront-layout','1384149808','ACTIVE',1,'2013-11-11 06:03:28','2013-11-11 06:03:30'),(55,'carefront-layout','1384150007','ACTIVE',1,'2013-11-11 06:06:48','2013-11-11 06:06:48'),(56,'carefront-client-layout','1384150021','ACTIVE',1,'2013-11-11 06:07:01','2013-11-11 06:07:03'),(57,'carefront-layout','1384268515','ACTIVE',1,'2013-11-12 15:01:55','2013-11-12 15:01:55'),(58,'carefront-client-layout','1384268525','ACTIVE',1,'2013-11-12 15:02:05','2013-11-12 15:02:05'),(59,'carefront-layout','1384270446','ACTIVE',1,'2013-11-12 15:34:06','2013-11-12 15:34:06'),(60,'carefront-client-layout','1384270457','ACTIVE',1,'2013-11-12 15:34:17','2013-11-12 15:34:17'),(61,'carefront-layout','1384270477','ACTIVE',1,'2013-11-12 15:34:37','2013-11-12 15:34:38'),(62,'carefront-layout','1384270479','ACTIVE',1,'2013-11-12 15:34:39','2013-11-12 15:34:39'),(63,'carefront-client-layout','1384270487','ACTIVE',1,'2013-11-12 15:34:47','2013-11-12 15:34:48'),(64,'carefront-client-layout','1384270489','ACTIVE',1,'2013-11-12 15:34:49','2013-11-12 15:34:49'),(65,'carefront-layout','1384270684','ACTIVE',1,'2013-11-12 15:38:04','2013-11-12 15:38:04'),(66,'carefront-client-layout','1384270694','ACTIVE',1,'2013-11-12 15:38:14','2013-11-12 15:38:15'),(67,'carefront-layout','1384270742','ACTIVE',1,'2013-11-12 15:39:02','2013-11-12 15:39:03'),(68,'carefront-client-layout','1384270751','ACTIVE',1,'2013-11-12 15:39:12','2013-11-12 15:39:12'),(69,'carefront-layout','1384275729','ACTIVE',1,'2013-11-12 17:02:09','2013-11-12 17:02:09'),(70,'carefront-client-layout','1384275739','ACTIVE',1,'2013-11-12 17:02:19','2013-11-12 17:02:19'),(71,'carefront-layout','1384275838','ACTIVE',1,'2013-11-12 17:03:58','2013-11-12 17:03:58'),(72,'carefront-client-layout','1384275847','ACTIVE',1,'2013-11-12 17:04:07','2013-11-12 17:04:07'),(73,'carefront-layout','1384276508','ACTIVE',1,'2013-11-12 17:15:08','2013-11-12 17:15:09'),(74,'carefront-client-layout','1384276518','ACTIVE',1,'2013-11-12 17:15:19','2013-11-12 17:15:19'),(75,'carefront-layout','1384285001','ACTIVE',1,'2013-11-12 19:36:41','2013-11-12 19:36:42'),(76,'carefront-client-layout','1384285011','ACTIVE',1,'2013-11-12 19:36:51','2013-11-12 19:36:51'),(77,'carefront-cases','19/51','CREATING',1,'2013-11-12 23:11:19','0000-00-00 00:00:00'),(78,'carefront-cases','19/52','CREATING',1,'2013-11-12 23:14:52','0000-00-00 00:00:00'),(79,'carefront-cases','19/54','CREATING',2,'2013-11-12 23:17:17','0000-00-00 00:00:00'),(80,'carefront-cases','19/55','ACTIVE',2,'2013-11-12 23:19:52','2013-11-12 23:19:53'),(81,'carefront-cases','19/56','ACTIVE',2,'2013-11-12 23:20:53','2013-11-12 23:20:53'),(82,'carefront-cases','19/57.1','CREATING',1,'2013-11-13 00:28:26','0000-00-00 00:00:00'),(83,'carefront-cases','19/0.1','CREATING',1,'2013-11-13 00:28:54','0000-00-00 00:00:00'),(84,'carefront-cases','19/59.1','CREATING',1,'2013-11-13 02:32:21','0000-00-00 00:00:00'),(85,'carefront-cases','19/60.1','ACTIVE',2,'2013-11-13 02:33:23','2013-11-13 02:33:24'),(86,'carefront-cases','19/0.1','ACTIVE',2,'2013-11-13 02:36:29','2013-11-13 02:36:31'),(87,'carefront-cases','19/0.1','CREATING',2,'2013-11-15 01:28:17','0000-00-00 00:00:00'),(90,'carefront-cases','19/65.1','ACTIVE',2,'2013-11-15 18:57:49','2013-11-15 18:57:55'),(91,'carefront-cases','6/66.out','ACTIVE',2,'2013-11-15 23:28:54','2013-11-15 23:28:55'),(92,'carefront-cases','6/67.out','ACTIVE',2,'2013-11-15 23:29:21','2013-11-15 23:29:25'),(93,'carefront-cases','6/68.out','ACTIVE',2,'2013-11-15 23:30:45','2013-11-15 23:30:49'),(94,'carefront-cases','6/69.out','ACTIVE',2,'2013-11-15 23:31:03','2013-11-15 23:31:05'),(95,'carefront-cases','6/70.out','ACTIVE',2,'2013-11-16 01:32:12','2013-11-16 01:32:13'),(96,'carefront-cases','6/71.out','ACTIVE',2,'2013-11-16 01:32:42','2013-11-16 01:32:43'),(97,'carefront-cases','6/72.out','ACTIVE',2,'2013-11-16 01:34:39','2013-11-16 01:34:39'),(98,'carefront-cases','6/73.out','ACTIVE',2,'2013-11-16 01:34:50','2013-11-16 01:34:51'),(99,'carefront-cases','6/74.out','ACTIVE',2,'2013-11-16 01:34:57','2013-11-16 01:34:57'),(100,'carefront-cases','48/75.out','ACTIVE',2,'2013-11-16 01:47:49','2013-11-16 01:47:51'),(101,'carefront-cases','48/76.out','ACTIVE',2,'2013-11-16 01:49:33','2013-11-16 01:49:33'),(102,'carefront-cases','50/77.out','ACTIVE',2,'2013-11-16 22:16:18','2013-11-16 22:16:19'),(103,'carefront-cases','50/0.out','ACTIVE',2,'2013-11-16 23:55:02','2013-11-16 23:55:03'),(104,'carefront-cases','50/83.out','ACTIVE',2,'2013-11-16 23:56:03','2013-11-16 23:56:04'),(105,'carefront-layout','1384648241','ACTIVE',1,'2013-11-17 00:30:41','2013-11-17 00:30:41'),(106,'carefront-client-layout','1384648251','ACTIVE',1,'2013-11-17 00:30:51','2013-11-17 00:30:52'),(107,'carefront-layout','1384648263','ACTIVE',1,'2013-11-17 00:31:07','2013-11-17 00:31:08'),(108,'carefront-client-layout','1384648276','ACTIVE',1,'2013-11-17 00:31:20','2013-11-17 00:31:20'),(109,'carefront-layout','1384649283','ACTIVE',1,'2013-11-17 00:48:07','2013-11-17 00:48:07'),(110,'carefront-client-layout','1384649297','ACTIVE',1,'2013-11-17 00:48:21','2013-11-17 00:48:22'),(111,'carefront-layout','1384716307','ACTIVE',1,'2013-11-17 19:25:07','2013-11-17 19:25:08'),(112,'carefront-client-layout','1384716322','ACTIVE',1,'2013-11-17 19:25:23','2013-11-17 19:25:23'),(113,'carefront-layout','1384716486','ACTIVE',1,'2013-11-17 19:28:06','2013-11-17 19:28:07'),(114,'carefront-client-layout','1384716500','ACTIVE',1,'2013-11-17 19:28:21','2013-11-17 19:28:22'),(115,'carefront-layout','1384716950','ACTIVE',1,'2013-11-17 19:35:51','2013-11-17 19:35:52'),(116,'carefront-client-layout','1384716965','ACTIVE',1,'2013-11-17 19:36:05','2013-11-17 19:36:06'),(117,'carefront-cases','53/96.out','ACTIVE',2,'2013-11-18 06:17:55','2013-11-18 06:17:57'),(118,'carefront-cases','53/97.out','ACTIVE',2,'2013-11-18 19:34:45','2013-11-18 19:34:47'),(119,'carefront-layout','1384910982','ACTIVE',1,'2013-11-20 01:29:41','2013-11-20 01:29:42'),(120,'carefront-layout','1384911003','ACTIVE',1,'2013-11-20 01:30:03','2013-11-20 01:30:04'),(121,'carefront-client-layout','1384911006','ACTIVE',1,'2013-11-20 01:30:06','2013-11-20 01:30:07'),(122,'carefront-layout','1384911441','ACTIVE',1,'2013-11-20 01:37:22','2013-11-20 01:37:24'),(123,'carefront-client-layout','1384911488','ACTIVE',1,'2013-11-20 01:38:08','2013-11-20 01:38:09'),(124,'carefront-layout','1384981429','ACTIVE',1,'2013-11-20 21:03:49','2013-11-20 21:03:50'),(125,'carefront-client-layout','1384981442','ACTIVE',1,'2013-11-20 21:04:02','2013-11-20 21:04:02'),(126,'carefront-cases','84/412.out','CREATING',1,'2013-11-23 17:43:25','0000-00-00 00:00:00'),(127,'carefront-doctor-layout-useast','1385249163','CREATING',1,'2013-11-23 23:26:03','0000-00-00 00:00:00'),(128,'carefront-doctor-layout-useast','1385249388','CREATING',1,'2013-11-23 23:29:48','0000-00-00 00:00:00'),(129,'carefront-doctor-layout-useast','1385249461','CREATING',1,'2013-11-23 23:31:02','0000-00-00 00:00:00'),(130,'carefront-doctor-layout-useast','1385249539','CREATING',1,'2013-11-23 23:32:19','0000-00-00 00:00:00'),(131,'carefront-doctor-layout-useast','1385249735','ACTIVE',1,'2013-11-23 23:35:35','2013-11-23 23:35:36'),(132,'carefront-doctor-layout-useast','1385250120','ACTIVE',1,'2013-11-23 23:42:00','2013-11-23 23:42:02'),(133,'carefront-doctor-layout-useast','1385250840','ACTIVE',1,'2013-11-23 23:54:00','2013-11-23 23:54:01'),(134,'carefront-doctor-layout-useast','1385250968','ACTIVE',1,'2013-11-23 23:56:08','2013-11-23 23:56:09'),(135,'carefront-cases-useast','84/413.out','CREATING',1,'2013-11-24 01:28:29','0000-00-00 00:00:00'),(136,'carefront-cases-useast','84/414.out','ACTIVE',1,'2013-11-24 01:30:25','2013-11-24 01:30:28'),(137,'carefront-layout','1385258558','ACTIVE',1,'2013-11-24 02:02:38','2013-11-24 02:02:39'),(138,'carefront-client-layout','1385258560','ACTIVE',1,'2013-11-24 02:02:40','2013-11-24 02:02:41'),(139,'carefront-layout','1385258716','ACTIVE',1,'2013-11-24 02:05:16','2013-11-24 02:05:17'),(140,'carefront-client-layout','1385258718','ACTIVE',1,'2013-11-24 02:05:18','2013-11-24 02:05:19'),(141,'carefront-layout','1385258945','ACTIVE',1,'2013-11-24 02:09:05','2013-11-24 02:09:06'),(142,'carefront-client-layout','1385258969','ACTIVE',1,'2013-11-24 02:09:29','2013-11-24 02:09:30'),(143,'carefront-layout','1385259067','ACTIVE',1,'2013-11-24 02:11:07','2013-11-24 02:11:09'),(144,'carefront-client-layout','1385259093','ACTIVE',1,'2013-11-24 02:11:34','2013-11-24 02:11:35'),(145,'carefront-layout','1385259622','ACTIVE',1,'2013-11-24 02:20:23','2013-11-24 02:20:23'),(146,'carefront-client-layout','1385259659','ACTIVE',1,'2013-11-24 02:21:00','2013-11-24 02:21:01'),(147,'carefront-doctor-layout-useast','1385260577','ACTIVE',1,'2013-11-24 02:36:17','2013-11-24 02:36:18'),(148,'carefront-doctor-layout-useast','1385260584','ACTIVE',1,'2013-11-24 02:36:24','2013-11-24 02:36:25'),(149,'carefront-doctor-visual-layout-useast','1385261177','ACTIVE',1,'2013-11-24 02:46:17','2013-11-24 02:46:18'),(150,'carefront-doctor-layout-useast','1385261184','ACTIVE',1,'2013-11-24 02:46:24','2013-11-24 02:46:25'),(151,'carefront-doctor-visual-layout-useast','1385332622','ACTIVE',1,'2013-11-24 22:37:03','2013-11-24 22:37:03'),(152,'carefront-doctor-layout-useast','1385332629','ACTIVE',1,'2013-11-24 22:37:10','2013-11-24 22:37:10'),(153,'carefront-cases-useast','84/415.out','ACTIVE',1,'2013-11-24 22:38:54','2013-11-24 22:38:57'),(154,'carefront-doctor-visual-layout-useast','1385335170','ACTIVE',1,'2013-11-24 23:19:30','2013-11-24 23:19:31'),(155,'carefront-doctor-layout-useast','1385335176','ACTIVE',1,'2013-11-24 23:19:36','2013-11-24 23:19:37'),(156,'carefront-doctor-visual-layout-useast','1385335246','ACTIVE',1,'2013-11-24 23:20:47','2013-11-24 23:20:47'),(157,'carefront-doctor-layout-useast','1385335253','ACTIVE',1,'2013-11-24 23:20:53','2013-11-24 23:20:54'),(158,'carefront-doctor-visual-layout-useast','1385335611','ACTIVE',1,'2013-11-24 23:26:51','2013-11-24 23:26:52'),(159,'carefront-doctor-layout-useast','1385335617','ACTIVE',1,'2013-11-24 23:26:58','2013-11-24 23:26:59'),(160,'carefront-cases-useast','86/416.out','ACTIVE',1,'2013-11-25 05:55:06','2013-11-25 05:55:08'),(161,'carefront-cases-useast','88/417.out','ACTIVE',1,'2013-11-25 22:42:41','2013-11-25 22:42:42');
/*!40000 ALTER TABLE `object_storage` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `layout_version`
--

DROP TABLE IF EXISTS `layout_version`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `layout_version` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `object_storage_id` int(10) unsigned NOT NULL,
  `syntax_version` int(10) unsigned NOT NULL,
  `health_condition_id` int(10) unsigned NOT NULL,
  `comment` varchar(600) DEFAULT NULL,
  `status` varchar(250) NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified_date` timestamp NOT NULL ON UPDATE CURRENT_TIMESTAMP,
  `role` varchar(250) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `object_storage_id` (`object_storage_id`,`syntax_version`,`health_condition_id`,`status`),
  KEY `treatment_id` (`health_condition_id`),
  CONSTRAINT `layout_version_ibfk_1` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`),
  CONSTRAINT `layout_version_ibfk_2` FOREIGN KEY (`object_storage_id`) REFERENCES `object_storage` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=63 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `layout_version`
--

LOCK TABLES `layout_version` WRITE;
/*!40000 ALTER TABLE `layout_version` DISABLE KEYS */;
INSERT INTO `layout_version` VALUES (15,33,1,1,'automatically generated','DEPCRECATED','2013-11-08 19:13:06','2013-11-23 23:08:35','PATIENT'),(16,35,1,1,'automatically generated','DEPCRECATED','2013-11-08 19:13:06','2013-11-23 23:08:35','PATIENT'),(17,37,1,1,'automatically generated','DEPCRECATED','2013-11-08 19:13:06','2013-11-23 23:08:35','PATIENT'),(18,39,1,1,'automatically generated','DEPCRECATED','2013-11-08 19:13:06','2013-11-23 23:08:35','PATIENT'),(19,41,1,1,'automatically generated','DEPCRECATED','2013-11-08 19:13:06','2013-11-23 23:08:35','PATIENT'),(20,43,1,1,'automatically generated','DEPCRECATED','2013-11-08 19:21:07','2013-11-23 23:08:35','PATIENT'),(21,45,1,1,'automatically generated','DEPCRECATED','2013-11-11 05:46:47','2013-11-23 23:08:35','PATIENT'),(22,47,1,1,'automatically generated','DEPCRECATED','2013-11-11 05:57:25','2013-11-23 23:08:35','PATIENT'),(23,49,1,1,'automatically generated','CREATING','2013-11-11 05:58:19','2013-11-23 23:08:35','PATIENT'),(24,50,1,1,'automatically generated','DEPCRECATED','2013-11-11 05:58:34','2013-11-23 23:08:35','PATIENT'),(25,52,1,1,'automatically generated','DEPCRECATED','2013-11-11 06:02:11','2013-11-23 23:08:35','PATIENT'),(26,55,1,1,'automatically generated','DEPCRECATED','2013-11-11 06:06:49','2013-11-23 23:08:35','PATIENT'),(27,57,1,1,'automatically generated','DEPCRECATED','2013-11-12 15:01:56','2013-11-23 23:08:35','PATIENT'),(28,59,1,1,'automatically generated','DEPCRECATED','2013-11-12 15:34:07','2013-11-23 23:08:35','PATIENT'),(29,61,1,1,'automatically generated','DEPCRECATED','2013-11-12 15:34:38','2013-11-23 23:08:35','PATIENT'),(30,62,1,1,'automatically generated','DEPCRECATED','2013-11-12 15:34:40','2013-11-23 23:08:35','PATIENT'),(31,65,1,1,'automatically generated','DEPCRECATED','2013-11-12 15:38:05','2013-11-23 23:08:35','PATIENT'),(32,67,1,1,'automatically generated','DEPCRECATED','2013-11-12 15:39:03','2013-11-23 23:08:35','PATIENT'),(33,69,1,1,'automatically generated','DEPCRECATED','2013-11-12 17:02:09','2013-11-23 23:08:35','PATIENT'),(34,71,1,1,'automatically generated','DEPCRECATED','2013-11-12 17:03:58','2013-11-23 23:08:35','PATIENT'),(35,73,1,1,'automatically generated','DEPCRECATED','2013-11-12 17:15:09','2013-11-23 23:08:35','PATIENT'),(36,75,1,1,'automatically generated','DEPCRECATED','2013-11-12 19:36:42','2013-11-23 23:08:35','PATIENT'),(37,105,1,1,'automatically generated','DEPCRECATED','2013-11-17 00:30:41','2013-11-23 23:08:35','PATIENT'),(38,107,1,1,'automatically generated','DEPCRECATED','2013-11-17 00:31:08','2013-11-23 23:08:35','PATIENT'),(39,109,1,1,'automatically generated','DEPCRECATED','2013-11-17 00:48:08','2013-11-23 23:08:35','PATIENT'),(40,111,1,1,'automatically generated','DEPCRECATED','2013-11-17 19:25:08','2013-11-23 23:08:35','PATIENT'),(41,113,1,1,'automatically generated','DEPCRECATED','2013-11-17 19:28:07','2013-11-23 23:08:35','PATIENT'),(42,115,1,1,'automatically generated','DEPCRECATED','2013-11-17 19:35:52','2013-11-23 23:08:35','PATIENT'),(43,119,1,1,'automatically generated','CREATING','2013-11-20 01:29:43','2013-11-23 23:08:35','PATIENT'),(44,120,1,1,'automatically generated','DEPCRECATED','2013-11-20 01:30:04','2013-11-23 23:08:35','PATIENT'),(45,122,1,1,'automatically generated','DEPCRECATED','2013-11-20 01:37:27','2013-11-23 23:08:35','PATIENT'),(46,124,1,1,'automatically generated','DEPCRECATED','2013-11-20 21:03:50','2013-11-24 02:02:41','PATIENT'),(48,131,1,1,'automatically generated','CREATING','2013-11-23 23:35:36','0000-00-00 00:00:00','DOCTOR'),(49,132,1,1,'automatically generated','CREATING','2013-11-23 23:42:03','0000-00-00 00:00:00','DOCTOR'),(50,133,1,1,'automatically generated','CREATING','2013-11-23 23:54:01','0000-00-00 00:00:00','DOCTOR'),(51,134,1,1,'automatically generated','DEPCRECATED','2013-11-23 23:56:09','2013-11-24 02:36:25','DOCTOR'),(52,137,1,1,'automatically generated','DEPCRECATED','2013-11-24 02:02:39','2013-11-24 02:05:20','PATIENT'),(53,139,1,1,'automatically generated','DEPCRECATED','2013-11-24 02:05:17','2013-11-24 02:09:31','PATIENT'),(54,141,1,1,'automatically generated','DEPCRECATED','2013-11-24 02:09:07','2013-11-24 02:11:36','PATIENT'),(55,143,1,1,'automatically generated','DEPCRECATED','2013-11-24 02:11:10','2013-11-24 02:21:03','PATIENT'),(56,145,1,1,'automatically generated','ACTIVE','2013-11-24 02:20:24','2013-11-24 02:21:03','PATIENT'),(57,147,1,1,'automatically generated','DEPCRECATED','2013-11-24 02:36:19','2013-11-24 02:46:25','DOCTOR'),(58,149,1,1,'automatically generated','DEPCRECATED','2013-11-24 02:46:18','2013-11-24 22:37:11','DOCTOR'),(59,151,1,1,'automatically generated','DEPCRECATED','2013-11-24 22:37:04','2013-11-24 23:19:38','DOCTOR'),(60,154,1,1,'automatically generated','DEPCRECATED','2013-11-24 23:19:31','2013-11-24 23:20:55','DOCTOR'),(61,156,1,1,'automatically generated','DEPCRECATED','2013-11-24 23:20:48','2013-11-24 23:26:59','DOCTOR'),(62,158,1,1,'automatically generated','ACTIVE','2013-11-24 23:26:52','2013-11-24 23:27:00','DOCTOR');
/*!40000 ALTER TABLE `layout_version` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `dr_layout_version`
--

DROP TABLE IF EXISTS `dr_layout_version`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_layout_version` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `object_storage_id` int(10) unsigned NOT NULL,
  `layout_version_id` int(10) unsigned NOT NULL,
  `status` varchar(250) NOT NULL,
  `modified_date` timestamp NOT NULL ON UPDATE CURRENT_TIMESTAMP,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `health_condition_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `layout_version_id` (`layout_version_id`),
  KEY `object_storage_id` (`object_storage_id`),
  KEY `health_condition_id` (`health_condition_id`),
  CONSTRAINT `dr_layout_version_ibfk_3` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`),
  CONSTRAINT `dr_layout_version_ibfk_1` FOREIGN KEY (`layout_version_id`) REFERENCES `layout_version` (`id`),
  CONSTRAINT `dr_layout_version_ibfk_2` FOREIGN KEY (`object_storage_id`) REFERENCES `object_storage` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `dr_layout_version`
--

LOCK TABLES `dr_layout_version` WRITE;
/*!40000 ALTER TABLE `dr_layout_version` DISABLE KEYS */;
INSERT INTO `dr_layout_version` VALUES (1,132,49,'CREATING','0000-00-00 00:00:00','2013-11-23 23:42:03',1),(2,133,50,'CREATING','0000-00-00 00:00:00','2013-11-23 23:54:02',1),(3,134,51,'DEPCRECATED','2013-11-24 02:36:26','2013-11-23 23:56:10',1),(4,148,57,'DEPCRECATED','2013-11-24 02:46:25','2013-11-24 02:36:25',1),(5,150,58,'DEPCRECATED','2013-11-24 22:37:11','2013-11-24 02:46:25',1),(6,152,59,'DEPCRECATED','2013-11-24 23:19:38','2013-11-24 22:37:11',1),(7,155,60,'DEPCRECATED','2013-11-24 23:20:55','2013-11-24 23:19:37',1),(8,157,61,'DEPCRECATED','2013-11-24 23:26:59','2013-11-24 23:20:54',1),(9,159,62,'ACTIVE','2013-11-24 23:27:00','2013-11-24 23:26:59',1);
/*!40000 ALTER TABLE `dr_layout_version` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2013-12-08 16:17:47
