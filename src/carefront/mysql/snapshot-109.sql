-- MySQL dump 10.13  Distrib 5.6.13, for osx10.8 (x86_64)
--
-- Host: dev-db-3.ccvrwjdx3gvp.us-east-1.rds.amazonaws.com    Database: database_20999
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
-- Table structure for table `account`
--

DROP TABLE IF EXISTS `account`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `account` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `email` varchar(250) DEFAULT NULL,
  `password` varbinary(250) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=94 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `advice`
--

DROP TABLE IF EXISTS `advice`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `advice` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_visit_id` int(10) unsigned NOT NULL,
  `dr_advice_point_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `patient_visit_id` (`patient_visit_id`),
  KEY `dr_advice_point_id` (`dr_advice_point_id`),
  CONSTRAINT `advice_ibfk_1` FOREIGN KEY (`patient_visit_id`) REFERENCES `patient_visit` (`id`),
  CONSTRAINT `advice_ibfk_2` FOREIGN KEY (`dr_advice_point_id`) REFERENCES `dr_advice_point` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `advice_point`
--

DROP TABLE IF EXISTS `advice_point`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `advice_point` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `text` varchar(150) NOT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
) ENGINE=InnoDB AUTO_INCREMENT=321 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `auth_token`
--

DROP TABLE IF EXISTS `auth_token`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `auth_token` (
  `token` varbinary(250) NOT NULL DEFAULT '',
  `account_id` int(10) unsigned NOT NULL,
  `created` timestamp NOT NULL,
  `expires` timestamp NOT NULL,
  PRIMARY KEY (`token`),
  KEY `account_id` (`account_id`),
  CONSTRAINT `auth_token_ibfk_1` FOREIGN KEY (`account_id`) REFERENCES `account` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `care_provider_state_elligibility`
--

DROP TABLE IF EXISTS `care_provider_state_elligibility`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `care_provider_state_elligibility` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `provider_role_id` int(10) unsigned NOT NULL,
  `provider_id` int(10) unsigned NOT NULL,
  `care_providing_state_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `provider_role_id` (`provider_role_id`),
  KEY `care_providing_state_id` (`care_providing_state_id`),
  CONSTRAINT `care_provider_state_elligibility_ibfk_1` FOREIGN KEY (`provider_role_id`) REFERENCES `provider_role` (`id`),
  CONSTRAINT `care_provider_state_elligibility_ibfk_2` FOREIGN KEY (`care_providing_state_id`) REFERENCES `care_providing_state` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `care_providing_state`
--

DROP TABLE IF EXISTS `care_providing_state`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `care_providing_state` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `state` varchar(100) NOT NULL,
  `health_condition_id` int(10) unsigned NOT NULL,
  `long_state` varchar(250) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `health_condition_id` (`health_condition_id`),
  CONSTRAINT `care_providing_state_ibfk_1` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `diagnosis_strength`
--

DROP TABLE IF EXISTS `diagnosis_strength`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `diagnosis_strength` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `tag` varchar(250) NOT NULL,
  `diagnosis_type_id` int(10) unsigned NOT NULL,
  `strength_title_text_id` int(10) unsigned NOT NULL,
  `strength_description_text_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `diagnosis_type_id` (`diagnosis_type_id`),
  KEY `strength_title_text_id` (`strength_title_text_id`),
  KEY `strength_description_text_id` (`strength_description_text_id`),
  CONSTRAINT `diagnosis_strength_ibfk_1` FOREIGN KEY (`diagnosis_type_id`) REFERENCES `diagnosis_type` (`id`),
  CONSTRAINT `diagnosis_strength_ibfk_2` FOREIGN KEY (`strength_title_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `diagnosis_strength_ibfk_3` FOREIGN KEY (`strength_description_text_id`) REFERENCES `app_text` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `diagnosis_summary`
--

DROP TABLE IF EXISTS `diagnosis_summary`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `diagnosis_summary` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_visit_id` int(10) unsigned NOT NULL,
  `doctor_id` int(10) unsigned NOT NULL,
  `summary` varchar(600) NOT NULL,
  `status` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_visit_id` (`patient_visit_id`),
  KEY `doctor_id` (`doctor_id`),
  CONSTRAINT `diagnosis_summary_ibfk_1` FOREIGN KEY (`patient_visit_id`) REFERENCES `patient_visit` (`id`),
  CONSTRAINT `diagnosis_summary_ibfk_2` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `diagnosis_type`
--

DROP TABLE IF EXISTS `diagnosis_type`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `diagnosis_type` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `tag` varchar(250) NOT NULL,
  `diagnosis_text_id` int(10) unsigned NOT NULL,
  `health_condition_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `diagnosis_text_id` (`diagnosis_text_id`),
  KEY `health_condition_id` (`health_condition_id`),
  CONSTRAINT `diagnosis_type_ibfk_1` FOREIGN KEY (`diagnosis_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `diagnosis_type_ibfk_2` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
-- Table structure for table `doctor`
--

DROP TABLE IF EXISTS `doctor`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `doctor` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `first_name` varchar(250) NOT NULL,
  `last_name` varchar(250) NOT NULL,
  `gender` varchar(250) NOT NULL,
  `dob` date NOT NULL,
  `account_id` int(10) unsigned NOT NULL,
  `dea_number` varchar(250) DEFAULT NULL,
  `npi_number` varchar(250) DEFAULT NULL,
  `status` varchar(250) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `account_id` (`account_id`),
  CONSTRAINT `doctor_ibfk_1` FOREIGN KEY (`account_id`) REFERENCES `account` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `doctor_queue`
--

DROP TABLE IF EXISTS `doctor_queue`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `doctor_queue` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `doctor_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  `event_type` varchar(100) NOT NULL,
  `enqueue_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `completed_date` timestamp NULL DEFAULT NULL,
  `item_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `doctor_id` (`doctor_id`),
  CONSTRAINT `doctor_queue_ibfk_1` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dr_advice_point`
--

DROP TABLE IF EXISTS `dr_advice_point`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_advice_point` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `text` varchar(150) NOT NULL,
  `doctor_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `doctor_id` (`doctor_id`),
  CONSTRAINT `dr_advice_point_ibfk_1` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dr_drug_supplemental_instruction`
--

DROP TABLE IF EXISTS `dr_drug_supplemental_instruction`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_drug_supplemental_instruction` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `doctor_id` int(10) unsigned NOT NULL,
  `text` varchar(150) NOT NULL,
  `drug_name_id` int(10) unsigned NOT NULL,
  `drug_form_id` int(10) unsigned DEFAULT NULL,
  `drug_route_id` int(10) unsigned DEFAULT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `drug_supplemental_instruction_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `drug_form_id` (`drug_form_id`),
  KEY `drug_route_id` (`drug_route_id`),
  KEY `drug_name_id` (`drug_name_id`),
  KEY `doctor_id` (`doctor_id`),
  KEY `drug_supplemental_instruction_id` (`drug_supplemental_instruction_id`),
  CONSTRAINT `dr_drug_supplemental_instruction_ibfk_1` FOREIGN KEY (`drug_form_id`) REFERENCES `drug_form` (`id`),
  CONSTRAINT `dr_drug_supplemental_instruction_ibfk_2` FOREIGN KEY (`drug_route_id`) REFERENCES `drug_route` (`id`),
  CONSTRAINT `dr_drug_supplemental_instruction_ibfk_3` FOREIGN KEY (`drug_name_id`) REFERENCES `drug_name` (`id`),
  CONSTRAINT `dr_drug_supplemental_instruction_ibfk_4` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`),
  CONSTRAINT `dr_drug_supplemental_instruction_ibfk_5` FOREIGN KEY (`drug_supplemental_instruction_id`) REFERENCES `drug_supplemental_instruction` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dr_drug_supplemental_instruction_selected_state`
--

DROP TABLE IF EXISTS `dr_drug_supplemental_instruction_selected_state`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_drug_supplemental_instruction_selected_state` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `drug_name_id` int(10) unsigned NOT NULL,
  `drug_form_id` int(10) unsigned NOT NULL,
  `drug_route_id` int(10) unsigned NOT NULL,
  `doctor_id` int(10) unsigned NOT NULL,
  `dr_drug_supplemental_instruction_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `doctor_id` (`doctor_id`),
  KEY `drug_form_id` (`drug_form_id`),
  KEY `drug_route_id` (`drug_route_id`),
  KEY `dr_drug_supplemental_instruction_id` (`dr_drug_supplemental_instruction_id`),
  KEY `drug_name_id` (`drug_name_id`,`drug_form_id`,`drug_route_id`,`doctor_id`,`dr_drug_supplemental_instruction_id`),
  CONSTRAINT `dr_drug_supplemental_instruction_selected_state_ibfk_1` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`),
  CONSTRAINT `dr_drug_supplemental_instruction_selected_state_ibfk_2` FOREIGN KEY (`drug_name_id`) REFERENCES `drug_name` (`id`),
  CONSTRAINT `dr_drug_supplemental_instruction_selected_state_ibfk_3` FOREIGN KEY (`drug_form_id`) REFERENCES `drug_form` (`id`),
  CONSTRAINT `dr_drug_supplemental_instruction_selected_state_ibfk_4` FOREIGN KEY (`drug_route_id`) REFERENCES `drug_route` (`id`),
  CONSTRAINT `dr_drug_supplemental_instruction_selected_state_ibfk_5` FOREIGN KEY (`dr_drug_supplemental_instruction_id`) REFERENCES `dr_drug_supplemental_instruction` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
) ENGINE=InnoDB AUTO_INCREMENT=25 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dr_regimen_step`
--

DROP TABLE IF EXISTS `dr_regimen_step`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_regimen_step` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `text` varchar(150) NOT NULL,
  `drug_name_id` int(10) unsigned DEFAULT NULL,
  `drug_form_id` int(10) unsigned DEFAULT NULL,
  `drug_route_id` int(10) unsigned DEFAULT NULL,
  `doctor_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `drug_name_id` (`drug_name_id`),
  KEY `drug_form_id` (`drug_form_id`),
  KEY `drug_route_id` (`drug_route_id`),
  KEY `doctor_id` (`doctor_id`),
  CONSTRAINT `dr_regimen_step_ibfk_1` FOREIGN KEY (`drug_name_id`) REFERENCES `drug_name` (`id`),
  CONSTRAINT `dr_regimen_step_ibfk_2` FOREIGN KEY (`drug_form_id`) REFERENCES `drug_form` (`id`),
  CONSTRAINT `dr_regimen_step_ibfk_3` FOREIGN KEY (`drug_route_id`) REFERENCES `drug_route` (`id`),
  CONSTRAINT `dr_regimen_step_ibfk_4` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `drug_db_id`
--

DROP TABLE IF EXISTS `drug_db_id`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `drug_db_id` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `drug_db_id_tag` varchar(100) NOT NULL,
  `drug_db_id` int(10) unsigned NOT NULL,
  `treatment_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `treatment_id` (`treatment_id`),
  CONSTRAINT `drug_db_id_ibfk_1` FOREIGN KEY (`treatment_id`) REFERENCES `treatment` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `drug_form`
--

DROP TABLE IF EXISTS `drug_form`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `drug_form` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(150) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `drug_name`
--

DROP TABLE IF EXISTS `drug_name`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `drug_name` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(150) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `drug_route`
--

DROP TABLE IF EXISTS `drug_route`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `drug_route` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(150) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
-- Table structure for table `info_intake`
--

DROP TABLE IF EXISTS `info_intake`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `info_intake` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_visit_id` int(10) unsigned DEFAULT NULL,
  `question_id` int(10) unsigned NOT NULL,
  `potential_answer_id` int(10) unsigned DEFAULT NULL,
  `answer_text` varchar(600) DEFAULT NULL,
  `layout_version_id` int(10) unsigned NOT NULL,
  `answered_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `status` varchar(100) NOT NULL,
  `role_id` int(10) unsigned NOT NULL,
  `object_storage_id` int(10) unsigned DEFAULT NULL,
  `parent_info_intake_id` int(10) unsigned DEFAULT NULL,
  `summary_localized_text_id` int(10) unsigned DEFAULT NULL,
  `parent_question_id` int(10) unsigned DEFAULT NULL,
  `role` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_visit_id` (`patient_visit_id`),
  KEY `question_id` (`question_id`),
  KEY `potential_answer_id` (`potential_answer_id`),
  KEY `layout_version_id` (`layout_version_id`),
  KEY `patient_id` (`role_id`),
  KEY `object_storage_id` (`object_storage_id`),
  KEY `parent_info_intake_id` (`parent_info_intake_id`),
  KEY `summary_localized_text_id` (`summary_localized_text_id`),
  KEY `parent_question_id` (`parent_question_id`),
  CONSTRAINT `info_intake_ibfk_1` FOREIGN KEY (`patient_visit_id`) REFERENCES `patient_visit` (`id`),
  CONSTRAINT `info_intake_ibfk_2` FOREIGN KEY (`question_id`) REFERENCES `question` (`id`),
  CONSTRAINT `info_intake_ibfk_3` FOREIGN KEY (`potential_answer_id`) REFERENCES `potential_answer` (`id`),
  CONSTRAINT `info_intake_ibfk_4` FOREIGN KEY (`layout_version_id`) REFERENCES `layout_version` (`id`),
  CONSTRAINT `info_intake_ibfk_6` FOREIGN KEY (`object_storage_id`) REFERENCES `object_storage` (`id`),
  CONSTRAINT `info_intake_ibfk_7` FOREIGN KEY (`parent_info_intake_id`) REFERENCES `info_intake` (`id`),
  CONSTRAINT `info_intake_ibfk_8` FOREIGN KEY (`summary_localized_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `info_intake_ibfk_9` FOREIGN KEY (`parent_question_id`) REFERENCES `question` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=502 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
  `layout_purpose` varchar(250) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `object_storage_id` (`object_storage_id`,`syntax_version`,`health_condition_id`,`status`),
  KEY `treatment_id` (`health_condition_id`),
  CONSTRAINT `layout_version_ibfk_1` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`),
  CONSTRAINT `layout_version_ibfk_2` FOREIGN KEY (`object_storage_id`) REFERENCES `object_storage` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=143 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
) ENGINE=InnoDB AUTO_INCREMENT=326 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
) ENGINE=InnoDB AUTO_INCREMENT=378 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient`
--

DROP TABLE IF EXISTS `patient`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `first_name` varchar(500) NOT NULL,
  `last_name` varchar(500) NOT NULL,
  `dob` date NOT NULL,
  `gender` varchar(500) NOT NULL,
  `status` varchar(500) NOT NULL,
  `account_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `account_id` (`account_id`),
  CONSTRAINT `patient_ibfk_1` FOREIGN KEY (`account_id`) REFERENCES `account` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=91 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_address`
--

DROP TABLE IF EXISTS `patient_address`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_address` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_id` int(10) unsigned NOT NULL,
  `address_line_1` varchar(300) DEFAULT NULL,
  `address_line_2` varchar(300) DEFAULT NULL,
  `city` varchar(300) DEFAULT NULL,
  `state` varchar(300) DEFAULT NULL,
  `zip_code` varchar(100) DEFAULT NULL,
  `address_type` varchar(100) DEFAULT NULL,
  `status` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_id` (`patient_id`),
  CONSTRAINT `patient_address_ibfk_1` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_care_provider_assignment`
--

DROP TABLE IF EXISTS `patient_care_provider_assignment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_care_provider_assignment` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `provider_role_id` int(10) unsigned NOT NULL,
  `provider_id` int(10) unsigned NOT NULL,
  `assignment_group_id` int(10) unsigned NOT NULL,
  `status` varchar(250) NOT NULL,
  `patient_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `assignment_group_id` (`assignment_group_id`),
  KEY `provider_role_id` (`provider_role_id`),
  KEY `patient_id` (`patient_id`),
  CONSTRAINT `patient_care_provider_assignment_ibfk_4` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`),
  CONSTRAINT `patient_care_provider_assignment_ibfk_2` FOREIGN KEY (`assignment_group_id`) REFERENCES `patient_care_provider_group` (`id`),
  CONSTRAINT `patient_care_provider_assignment_ibfk_3` FOREIGN KEY (`provider_role_id`) REFERENCES `provider_role` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_care_provider_group`
--

DROP TABLE IF EXISTS `patient_care_provider_group`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_care_provider_group` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified_date` timestamp NOT NULL ON UPDATE CURRENT_TIMESTAMP,
  `status` varchar(250) NOT NULL,
  `patient_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_id` (`patient_id`),
  CONSTRAINT `patient_care_provider_group_ibfk_1` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_diagnosis`
--

DROP TABLE IF EXISTS `patient_diagnosis`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_diagnosis` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `diagnosis_type_id` int(10) unsigned NOT NULL,
  `diagnosis_strength_id` int(10) unsigned NOT NULL,
  `patient_visit_id` int(10) unsigned NOT NULL,
  `diagnosis_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `status` varchar(250) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `diagnosis_type_id` (`diagnosis_type_id`),
  KEY `diagnosis_strength_id` (`diagnosis_strength_id`),
  KEY `patient_visit_id` (`patient_visit_id`),
  CONSTRAINT `patient_diagnosis_ibfk_1` FOREIGN KEY (`diagnosis_type_id`) REFERENCES `diagnosis_type` (`id`),
  CONSTRAINT `patient_diagnosis_ibfk_2` FOREIGN KEY (`diagnosis_strength_id`) REFERENCES `diagnosis_strength` (`id`),
  CONSTRAINT `patient_diagnosis_ibfk_3` FOREIGN KEY (`patient_visit_id`) REFERENCES `patient_visit` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
) ENGINE=InnoDB AUTO_INCREMENT=104 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_location`
--

DROP TABLE IF EXISTS `patient_location`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_location` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_id` int(10) unsigned NOT NULL,
  `zip_code` varchar(100) NOT NULL,
  `city` varchar(150) DEFAULT NULL,
  `state` varchar(150) DEFAULT NULL,
  `status` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_id` (`patient_id`),
  CONSTRAINT `patient_location_ibfk_1` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_phone`
--

DROP TABLE IF EXISTS `patient_phone`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_phone` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_id` int(10) unsigned NOT NULL,
  `phone` varchar(100) NOT NULL,
  `phone_type` varchar(100) NOT NULL,
  `status` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_id` (`patient_id`),
  CONSTRAINT `patient_phone_ibfk_1` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_visit`
--

DROP TABLE IF EXISTS `patient_visit`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_visit` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_id` int(10) unsigned NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `closed_date` timestamp NULL DEFAULT NULL,
  `health_condition_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  `layout_version_id` int(10) unsigned NOT NULL,
  `submitted_date` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_id` (`patient_id`),
  KEY `treatment_id` (`health_condition_id`),
  KEY `layout_version_id` (`layout_version_id`),
  CONSTRAINT `patient_visit_ibfk_1` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`),
  CONSTRAINT `patient_visit_ibfk_2` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`),
  CONSTRAINT `patient_visit_ibfk_3` FOREIGN KEY (`layout_version_id`) REFERENCES `layout_version` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=89 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_visit_care_provider_assignment`
--

DROP TABLE IF EXISTS `patient_visit_care_provider_assignment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_visit_care_provider_assignment` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_visit_id` int(10) unsigned NOT NULL,
  `provider_role_id` int(10) unsigned NOT NULL,
  `provider_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  `assignment_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `patient_visit_id` (`patient_visit_id`),
  KEY `provider_role` (`provider_role_id`),
  CONSTRAINT `patient_visit_care_provider_assignment_ibfk_1` FOREIGN KEY (`patient_visit_id`) REFERENCES `patient_visit` (`id`),
  CONSTRAINT `patient_visit_care_provider_assignment_ibfk_2` FOREIGN KEY (`provider_role_id`) REFERENCES `provider_role` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_visit_follow_up`
--

DROP TABLE IF EXISTS `patient_visit_follow_up`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_visit_follow_up` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_visit_id` int(10) unsigned NOT NULL,
  `doctor_id` int(10) unsigned NOT NULL,
  `follow_up_date` date NOT NULL,
  `follow_up_value` int(10) unsigned NOT NULL,
  `follow_up_unit` varchar(100) NOT NULL,
  `status` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_visit_id` (`patient_visit_id`),
  KEY `doctor_id` (`doctor_id`),
  CONSTRAINT `patient_visit_follow_up_ibfk_1` FOREIGN KEY (`patient_visit_id`) REFERENCES `patient_visit` (`id`),
  CONSTRAINT `patient_visit_follow_up_ibfk_2` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
  `status` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `potential_outcome_tag` (`potential_answer_tag`),
  UNIQUE KEY `question_id_2` (`question_id`,`ordering`),
  KEY `otype_id` (`atype_id`),
  KEY `outcome_localized_text` (`answer_localized_text_id`),
  KEY `answer_summary_text_id` (`answer_summary_text_id`),
  CONSTRAINT `potential_answer_ibfk_1` FOREIGN KEY (`atype_id`) REFERENCES `answer_type` (`id`),
  CONSTRAINT `potential_answer_ibfk_2` FOREIGN KEY (`question_id`) REFERENCES `question` (`id`),
  CONSTRAINT `potential_answer_ibfk_3` FOREIGN KEY (`answer_summary_text_id`) REFERENCES `app_text` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=136 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `provider_role`
--

DROP TABLE IF EXISTS `provider_role`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `provider_role` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `provider_tag` varchar(250) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
  `formatted_field_tags` varchar(150) NOT NULL,
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
) ENGINE=InnoDB AUTO_INCREMENT=45 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `question_fields`
--

DROP TABLE IF EXISTS `question_fields`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `question_fields` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `question_field` varchar(250) NOT NULL,
  `question_id` int(10) unsigned NOT NULL,
  `app_text_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `question_id` (`question_id`),
  KEY `app_text_id` (`app_text_id`),
  KEY `question_field` (`question_field`,`question_id`),
  CONSTRAINT `question_fields_ibfk_1` FOREIGN KEY (`question_id`) REFERENCES `question` (`id`),
  CONSTRAINT `question_fields_ibfk_2` FOREIGN KEY (`app_text_id`) REFERENCES `app_text` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=57 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `regimen`
--

DROP TABLE IF EXISTS `regimen`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `regimen` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `regimen_type` varchar(150) NOT NULL,
  `patient_visit_id` int(10) unsigned NOT NULL,
  `dr_regimen_step_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `patient_visit_id` (`patient_visit_id`),
  KEY `dr_regimen_step_id` (`dr_regimen_step_id`),
  CONSTRAINT `regimen_ibfk_1` FOREIGN KEY (`patient_visit_id`) REFERENCES `patient_visit` (`id`),
  CONSTRAINT `regimen_ibfk_2` FOREIGN KEY (`dr_regimen_step_id`) REFERENCES `dr_regimen_step` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `regimen_step`
--

DROP TABLE IF EXISTS `regimen_step`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `regimen_step` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `text` varchar(150) NOT NULL,
  `drug_name_id` int(10) unsigned DEFAULT NULL,
  `drug_form_id` int(10) unsigned DEFAULT NULL,
  `drug_route_id` int(10) unsigned DEFAULT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `drug_name_id` (`drug_name_id`),
  KEY `drug_form_id` (`drug_form_id`),
  KEY `drug_route_id` (`drug_route_id`),
  CONSTRAINT `regimen_step_ibfk_1` FOREIGN KEY (`drug_name_id`) REFERENCES `drug_name` (`id`),
  CONSTRAINT `regimen_step_ibfk_2` FOREIGN KEY (`drug_form_id`) REFERENCES `drug_form` (`id`),
  CONSTRAINT `regimen_step_ibfk_3` FOREIGN KEY (`drug_route_id`) REFERENCES `drug_route` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
-- Table structure for table `treatment`
--

DROP TABLE IF EXISTS `treatment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `treatment` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `treatment_plan_id` int(10) unsigned NOT NULL,
  `drug_internal_name` varchar(250) NOT NULL,
  `dispense_value` int(10) unsigned NOT NULL,
  `dispense_unit_id` int(10) unsigned NOT NULL,
  `refills` int(10) unsigned NOT NULL,
  `substitutions_allowed` tinyint(1) DEFAULT NULL,
  `days_supply` int(10) unsigned NOT NULL,
  `pharmacy_notes` varchar(150) DEFAULT NULL,
  `patient_instructions` varchar(150) NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `status` varchar(100) NOT NULL,
  `dosage_strength` varchar(250) NOT NULL,
  `type` varchar(150) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `treatment_plan_id` (`treatment_plan_id`),
  KEY `dispense_unit_id` (`dispense_unit_id`),
  CONSTRAINT `treatment_ibfk_4` FOREIGN KEY (`treatment_plan_id`) REFERENCES `treatment_plan` (`id`),
  CONSTRAINT `treatment_ibfk_3` FOREIGN KEY (`dispense_unit_id`) REFERENCES `dispense_unit` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `treatment_instructions`
--

DROP TABLE IF EXISTS `treatment_instructions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `treatment_instructions` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `treatment_id` int(10) unsigned NOT NULL,
  `dr_drug_instruction_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `treatment_id` (`treatment_id`),
  KEY `dr_drug_instruction_id` (`dr_drug_instruction_id`),
  CONSTRAINT `treatment_instructions_ibfk_2` FOREIGN KEY (`dr_drug_instruction_id`) REFERENCES `dr_drug_supplemental_instruction` (`id`),
  CONSTRAINT `treatment_instructions_ibfk_1` FOREIGN KEY (`treatment_id`) REFERENCES `treatment` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `treatment_plan`
--

DROP TABLE IF EXISTS `treatment_plan`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `treatment_plan` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `status` varchar(100) NOT NULL,
  `patient_visit_id` int(10) unsigned NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `patient_visit_id` (`patient_visit_id`),
  CONSTRAINT `treatment_plan_ibfk_1` FOREIGN KEY (`patient_visit_id`) REFERENCES `patient_visit` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2014-01-10 23:24:00
