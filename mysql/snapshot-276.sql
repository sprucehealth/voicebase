-- MySQL dump 10.13  Distrib 5.6.17, for osx10.9 (x86_64)
--
-- Host: 127.0.0.1    Database: database_8249
-- ------------------------------------------------------
-- Server version	5.6.17

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
  `role_type_id` int(10) unsigned NOT NULL,
  `registration_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `last_opened_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `email` (`email`),
  KEY `role_type_id` (`role_type_id`),
  CONSTRAINT `account_ibfk_1` FOREIGN KEY (`role_type_id`) REFERENCES `role_type` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=94 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `account_phone`
--

DROP TABLE IF EXISTS `account_phone`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `account_phone` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `account_id` int(10) unsigned NOT NULL,
  `phone` varchar(64) NOT NULL,
  `phone_type` varchar(32) NOT NULL,
  `status` varchar(32) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `account_id` (`account_id`),
  CONSTRAINT `account_phone_ibfk_1` FOREIGN KEY (`account_id`) REFERENCES `account` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `address`
--

DROP TABLE IF EXISTS `address`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `address` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `address_line_1` varchar(500) NOT NULL,
  `address_line_2` varchar(500) NOT NULL,
  `city` varchar(500) NOT NULL,
  `state` varchar(500) NOT NULL,
  `country` varchar(500) NOT NULL,
  `zip_code` varchar(500) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `advice`
--

DROP TABLE IF EXISTS `advice`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `advice` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `dr_advice_point_id` int(10) unsigned DEFAULT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `treatment_plan_id` int(10) unsigned NOT NULL,
  `text` varchar(150) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `dr_advice_point_id` (`dr_advice_point_id`),
  KEY `treatment_plan_id` (`treatment_plan_id`),
  CONSTRAINT `advice_ibfk_2` FOREIGN KEY (`dr_advice_point_id`) REFERENCES `dr_advice_point` (`id`),
  CONSTRAINT `advice_ibfk_3` FOREIGN KEY (`treatment_plan_id`) REFERENCES `treatment_plan` (`id`) ON DELETE CASCADE
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
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
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
) ENGINE=InnoDB AUTO_INCREMENT=18 DEFAULT CHARSET=utf8;
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
) ENGINE=InnoDB AUTO_INCREMENT=424 DEFAULT CHARSET=utf8;
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
  `created` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `expires` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (`token`),
  UNIQUE KEY `account_id_2` (`account_id`),
  KEY `account_id` (`account_id`),
  CONSTRAINT `auth_token_ibfk_1` FOREIGN KEY (`account_id`) REFERENCES `account` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `bank_account`
--

DROP TABLE IF EXISTS `bank_account`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `bank_account` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `account_id` int(10) unsigned NOT NULL,
  `stripe_recipient_id` varchar(128) NOT NULL,
  `default_account` tinyint(1) NOT NULL,
  `verified` tinyint(1) NOT NULL DEFAULT '0',
  `verify_amount_1` int(11) DEFAULT NULL,
  `verify_amount_2` int(11) DEFAULT NULL,
  `verify_transfer1_id` varchar(128) DEFAULT NULL,
  `verify_transfer2_id` varchar(128) DEFAULT NULL,
  `verify_expires` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `account_id` (`account_id`),
  CONSTRAINT `bank_account_ibfk_1` FOREIGN KEY (`account_id`) REFERENCES `account` (`id`)
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
  `role_type_id` int(10) unsigned NOT NULL,
  `provider_id` int(10) unsigned NOT NULL,
  `care_providing_state_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `provider_role_id` (`role_type_id`),
  KEY `care_providing_state_id` (`care_providing_state_id`),
  CONSTRAINT `care_provider_state_elligibility_ibfk_1` FOREIGN KEY (`role_type_id`) REFERENCES `role_type` (`id`),
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
-- Table structure for table `case_notification`
--

DROP TABLE IF EXISTS `case_notification`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `case_notification` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_case_id` int(10) unsigned NOT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `notification_type` varchar(100) NOT NULL,
  `uid` varchar(100) NOT NULL,
  `data` blob,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid` (`uid`),
  KEY `patient_case_id` (`patient_case_id`),
  CONSTRAINT `case_notification_ibfk_1` FOREIGN KEY (`patient_case_id`) REFERENCES `patient_case` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `communication_preference`
--

DROP TABLE IF EXISTS `communication_preference`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `communication_preference` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `account_id` int(10) unsigned NOT NULL,
  `communication_type` varchar(50) NOT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `status` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_id` (`account_id`,`communication_type`),
  CONSTRAINT `communication_preference_ibfk_1` FOREIGN KEY (`account_id`) REFERENCES `account` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `credit_card`
--

DROP TABLE IF EXISTS `credit_card`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `credit_card` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `third_party_card_id` varchar(100) DEFAULT NULL,
  `type` varchar(100) NOT NULL,
  `patient_id` int(10) unsigned NOT NULL,
  `address_id` int(10) unsigned NOT NULL,
  `is_default` tinyint(1) NOT NULL,
  `label` varchar(200) DEFAULT NULL,
  `status` varchar(100) NOT NULL,
  `fingerprint` varchar(200) DEFAULT NULL,
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `address_id` (`address_id`),
  KEY `patient_id` (`patient_id`),
  CONSTRAINT `credit_card_ibfk_1` FOREIGN KEY (`address_id`) REFERENCES `address` (`id`),
  CONSTRAINT `credit_card_ibfk_2` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
-- Table structure for table `dntf_mapping`
--

DROP TABLE IF EXISTS `dntf_mapping`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dntf_mapping` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `treatment_id` int(10) unsigned DEFAULT NULL,
  `unlinked_dntf_treatment_id` int(10) unsigned DEFAULT NULL,
  `rx_refill_request_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `treatment_id` (`treatment_id`),
  KEY `rx_refill_request_id` (`rx_refill_request_id`),
  KEY `unlinked_dntf_treatment_id` (`unlinked_dntf_treatment_id`),
  CONSTRAINT `dntf_mapping_ibfk_1` FOREIGN KEY (`treatment_id`) REFERENCES `treatment` (`id`),
  CONSTRAINT `dntf_mapping_ibfk_2` FOREIGN KEY (`rx_refill_request_id`) REFERENCES `rx_refill_request` (`id`),
  CONSTRAINT `dntf_mapping_ibfk_3` FOREIGN KEY (`unlinked_dntf_treatment_id`) REFERENCES `unlinked_dntf_treatment` (`id`)
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
  `account_id` int(10) unsigned NOT NULL,
  `dea_number` varchar(250) DEFAULT NULL,
  `npi_number` varchar(250) DEFAULT NULL,
  `status` varchar(250) NOT NULL,
  `clinician_id` int(10) unsigned DEFAULT NULL,
  `dob_month` int(10) unsigned DEFAULT NULL,
  `dob_year` int(10) unsigned DEFAULT NULL,
  `dob_day` int(10) unsigned DEFAULT NULL,
  `middle_name` varchar(100) DEFAULT NULL,
  `prefix` varchar(100) DEFAULT NULL,
  `suffix` varchar(100) DEFAULT NULL,
  `short_title` varchar(300) DEFAULT NULL,
  `long_title` varchar(300) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `account_id` (`account_id`),
  CONSTRAINT `doctor_ibfk_1` FOREIGN KEY (`account_id`) REFERENCES `account` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `doctor_address_selection`
--

DROP TABLE IF EXISTS `doctor_address_selection`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `doctor_address_selection` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `address_id` int(10) unsigned NOT NULL,
  `doctor_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `doctor_id` (`doctor_id`),
  KEY `address_id` (`address_id`),
  CONSTRAINT `doctor_address_selection_ibfk_1` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`),
  CONSTRAINT `doctor_address_selection_ibfk_2` FOREIGN KEY (`address_id`) REFERENCES `address` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `doctor_attribute`
--

DROP TABLE IF EXISTS `doctor_attribute`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `doctor_attribute` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `doctor_id` int(10) unsigned NOT NULL,
  `name` varchar(64) NOT NULL,
  `value` varchar(1024) NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `doctor_id` (`doctor_id`,`name`),
  CONSTRAINT `doctor_attribute_ibfk_1` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `doctor_medical_license`
--

DROP TABLE IF EXISTS `doctor_medical_license`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `doctor_medical_license` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `doctor_id` int(10) unsigned NOT NULL,
  `state` char(2) NOT NULL,
  `license_number` varchar(64) NOT NULL,
  `status` varchar(32) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `doctor_id` (`doctor_id`,`state`),
  CONSTRAINT `doctor_medical_license_ibfk_1` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
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
  `enqueue_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `completed_date` timestamp NULL DEFAULT NULL,
  `item_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `doctor_id` (`doctor_id`),
  CONSTRAINT `doctor_queue_ibfk_1` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `doctor_saved_case_message`
--

DROP TABLE IF EXISTS `doctor_saved_case_message`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `doctor_saved_case_message` (
  `doctor_id` int(10) unsigned NOT NULL,
  `message` text NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`doctor_id`),
  CONSTRAINT `doctor_saved_case_message_ibfk_1` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
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
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `source_id` int(10) unsigned DEFAULT NULL,
  `modified_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `doctor_id` (`doctor_id`),
  KEY `source_id` (`source_id`),
  KEY `doctor_id_2` (`doctor_id`),
  CONSTRAINT `dr_advice_point_ibfk_1` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`),
  CONSTRAINT `dr_advice_point_ibfk_2` FOREIGN KEY (`source_id`) REFERENCES `dr_advice_point` (`id`)
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
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
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
-- Table structure for table `dr_favorite_advice`
--

DROP TABLE IF EXISTS `dr_favorite_advice`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_favorite_advice` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `text` varchar(150) NOT NULL,
  `dr_advice_point_id` int(10) unsigned NOT NULL,
  `dr_favorite_treatment_plan_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `dr_advice_point_id` (`dr_advice_point_id`),
  KEY `dr_favorite_treatment_plan_id` (`dr_favorite_treatment_plan_id`),
  CONSTRAINT `dr_favorite_advice_ibfk_2` FOREIGN KEY (`dr_favorite_treatment_plan_id`) REFERENCES `dr_favorite_treatment_plan` (`id`) ON DELETE CASCADE,
  CONSTRAINT `dr_favorite_advice_ibfk_1` FOREIGN KEY (`dr_advice_point_id`) REFERENCES `dr_advice_point` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dr_favorite_patient_visit_follow_up`
--

DROP TABLE IF EXISTS `dr_favorite_patient_visit_follow_up`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_favorite_patient_visit_follow_up` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `follow_up_date` date NOT NULL,
  `follow_up_value` int(10) unsigned NOT NULL,
  `follow_up_unit` varchar(100) NOT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `status` varchar(100) NOT NULL,
  `dr_favorite_treatment_plan_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `dr_favorite_treatment_plan_id` (`dr_favorite_treatment_plan_id`),
  CONSTRAINT `dr_favorite_patient_visit_follow_up_ibfk_1` FOREIGN KEY (`dr_favorite_treatment_plan_id`) REFERENCES `dr_favorite_treatment_plan` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dr_favorite_regimen`
--

DROP TABLE IF EXISTS `dr_favorite_regimen`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_favorite_regimen` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `regimen_type` varchar(150) NOT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `text` varchar(150) NOT NULL,
  `dr_regimen_step_id` int(10) unsigned NOT NULL,
  `dr_favorite_treatment_plan_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `dr_favorite_treatment_plan_id` (`dr_favorite_treatment_plan_id`),
  KEY `dr_regimen_step_id` (`dr_regimen_step_id`),
  CONSTRAINT `dr_favorite_regimen_ibfk_3` FOREIGN KEY (`dr_favorite_treatment_plan_id`) REFERENCES `dr_favorite_treatment_plan` (`id`) ON DELETE CASCADE,
  CONSTRAINT `dr_favorite_regimen_ibfk_2` FOREIGN KEY (`dr_regimen_step_id`) REFERENCES `dr_regimen_step` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dr_favorite_treatment`
--

DROP TABLE IF EXISTS `dr_favorite_treatment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_favorite_treatment` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `dr_favorite_treatment_plan_id` int(10) unsigned NOT NULL,
  `drug_internal_name` varchar(250) NOT NULL,
  `dispense_value` decimal(21,10) NOT NULL,
  `dispense_unit_id` int(10) unsigned NOT NULL,
  `refills` int(10) unsigned NOT NULL,
  `substitutions_allowed` tinyint(4) DEFAULT NULL,
  `days_supply` int(10) unsigned DEFAULT NULL,
  `pharmacy_notes` varchar(150) DEFAULT NULL,
  `patient_instructions` varchar(150) NOT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `status` varchar(100) NOT NULL,
  `dosage_strength` varchar(250) NOT NULL,
  `type` varchar(150) NOT NULL,
  `drug_name_id` int(10) unsigned DEFAULT NULL,
  `drug_form_id` int(10) unsigned DEFAULT NULL,
  `drug_route_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `dr_favorite_treatment_plan_id` (`dr_favorite_treatment_plan_id`),
  KEY `dispense_unit_id` (`dispense_unit_id`),
  KEY `drug_name_id` (`drug_name_id`),
  KEY `drug_route_id` (`drug_route_id`),
  KEY `drug_form_id` (`drug_form_id`),
  CONSTRAINT `dr_favorite_treatment_ibfk_6` FOREIGN KEY (`dr_favorite_treatment_plan_id`) REFERENCES `dr_favorite_treatment_plan` (`id`) ON DELETE CASCADE,
  CONSTRAINT `dr_favorite_treatment_ibfk_2` FOREIGN KEY (`dispense_unit_id`) REFERENCES `dispense_unit` (`id`),
  CONSTRAINT `dr_favorite_treatment_ibfk_3` FOREIGN KEY (`drug_name_id`) REFERENCES `drug_name` (`id`),
  CONSTRAINT `dr_favorite_treatment_ibfk_4` FOREIGN KEY (`drug_route_id`) REFERENCES `drug_route` (`id`),
  CONSTRAINT `dr_favorite_treatment_ibfk_5` FOREIGN KEY (`drug_form_id`) REFERENCES `drug_form` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dr_favorite_treatment_drug_db_id`
--

DROP TABLE IF EXISTS `dr_favorite_treatment_drug_db_id`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_favorite_treatment_drug_db_id` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `drug_db_id` varchar(100) NOT NULL,
  `drug_db_id_tag` varchar(100) NOT NULL,
  `dr_favorite_treatment_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `dr_favorite_treatment_id` (`dr_favorite_treatment_id`),
  CONSTRAINT `dr_favorite_treatment_drug_db_id_ibfk_1` FOREIGN KEY (`dr_favorite_treatment_id`) REFERENCES `dr_favorite_treatment` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dr_favorite_treatment_plan`
--

DROP TABLE IF EXISTS `dr_favorite_treatment_plan`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_favorite_treatment_plan` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(600) NOT NULL,
  `doctor_id` int(10) unsigned NOT NULL,
  `modified_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `doctor_id` (`doctor_id`),
  KEY `doctor_id_2` (`doctor_id`),
  CONSTRAINT `dr_favorite_treatment_plan_ibfk_1` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
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
  `object_storage_id` int(10) unsigned DEFAULT NULL,
  `layout_version_id` int(10) unsigned NOT NULL,
  `status` varchar(250) NOT NULL,
  `modified_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `health_condition_id` int(10) unsigned NOT NULL,
  `layout_blob_storage_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `layout_version_id` (`layout_version_id`),
  KEY `object_storage_id` (`object_storage_id`),
  KEY `health_condition_id` (`health_condition_id`),
  KEY `layout_blob_storage_id` (`layout_blob_storage_id`),
  CONSTRAINT `dr_layout_version_ibfk_1` FOREIGN KEY (`layout_version_id`) REFERENCES `layout_version` (`id`),
  CONSTRAINT `dr_layout_version_ibfk_3` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`),
  CONSTRAINT `dr_layout_version_ibfk_4` FOREIGN KEY (`object_storage_id`) REFERENCES `object_storage` (`id`),
  CONSTRAINT `dr_layout_version_ibfk_5` FOREIGN KEY (`layout_blob_storage_id`) REFERENCES `layout_blob_storage` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=55 DEFAULT CHARSET=utf8;
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
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `source_id` int(10) unsigned DEFAULT NULL,
  `modified_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `drug_name_id` (`drug_name_id`),
  KEY `drug_form_id` (`drug_form_id`),
  KEY `drug_route_id` (`drug_route_id`),
  KEY `doctor_id` (`doctor_id`),
  KEY `source_id` (`source_id`),
  KEY `doctor_id_2` (`doctor_id`),
  CONSTRAINT `dr_regimen_step_ibfk_1` FOREIGN KEY (`drug_name_id`) REFERENCES `drug_name` (`id`),
  CONSTRAINT `dr_regimen_step_ibfk_2` FOREIGN KEY (`drug_form_id`) REFERENCES `drug_form` (`id`),
  CONSTRAINT `dr_regimen_step_ibfk_3` FOREIGN KEY (`drug_route_id`) REFERENCES `drug_route` (`id`),
  CONSTRAINT `dr_regimen_step_ibfk_4` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`),
  CONSTRAINT `dr_regimen_step_ibfk_5` FOREIGN KEY (`source_id`) REFERENCES `dr_regimen_step` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dr_treatment_template`
--

DROP TABLE IF EXISTS `dr_treatment_template`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_treatment_template` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(600) NOT NULL,
  `doctor_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  `drug_internal_name` varchar(250) NOT NULL,
  `dispense_value` decimal(21,10) NOT NULL,
  `dispense_unit_id` int(10) unsigned NOT NULL,
  `refills` int(10) unsigned NOT NULL,
  `substitutions_allowed` tinyint(4) NOT NULL,
  `days_supply` int(10) unsigned DEFAULT NULL,
  `pharmacy_notes` varchar(150) DEFAULT NULL,
  `patient_instructions` varchar(150) NOT NULL,
  `dosage_strength` varchar(250) NOT NULL,
  `type` varchar(150) NOT NULL,
  `drug_name_id` int(10) unsigned NOT NULL,
  `drug_form_id` int(10) unsigned DEFAULT NULL,
  `drug_route_id` int(10) unsigned DEFAULT NULL,
  `erx_sent_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `erx_id` int(10) unsigned DEFAULT NULL,
  `pharmacy_id` int(10) unsigned DEFAULT NULL,
  `erx_last_filled_date` timestamp(6) NULL DEFAULT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `doctor_id` (`doctor_id`),
  KEY `dispense_unit_id` (`dispense_unit_id`),
  KEY `drug_name_id` (`drug_name_id`),
  KEY `drug_route_id` (`drug_route_id`),
  KEY `drug_form_id` (`drug_form_id`),
  KEY `pharmacy_id` (`pharmacy_id`),
  CONSTRAINT `dr_treatment_template_ibfk_1` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`),
  CONSTRAINT `dr_treatment_template_ibfk_2` FOREIGN KEY (`dispense_unit_id`) REFERENCES `dispense_unit` (`id`),
  CONSTRAINT `dr_treatment_template_ibfk_3` FOREIGN KEY (`drug_name_id`) REFERENCES `drug_name` (`id`),
  CONSTRAINT `dr_treatment_template_ibfk_4` FOREIGN KEY (`drug_route_id`) REFERENCES `drug_route` (`id`),
  CONSTRAINT `dr_treatment_template_ibfk_5` FOREIGN KEY (`drug_form_id`) REFERENCES `drug_form` (`id`),
  CONSTRAINT `dr_treatment_template_ibfk_6` FOREIGN KEY (`pharmacy_id`) REFERENCES `pharmacy_selection` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dr_treatment_template_drug_db_id`
--

DROP TABLE IF EXISTS `dr_treatment_template_drug_db_id`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dr_treatment_template_drug_db_id` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `drug_db_id_tag` varchar(100) NOT NULL,
  `drug_db_id` varchar(100) NOT NULL,
  `dr_treatment_template_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `dr_treatment_template_id` (`dr_treatment_template_id`),
  CONSTRAINT `dr_treatment_template_drug_db_id_ibfk_1` FOREIGN KEY (`dr_treatment_template_id`) REFERENCES `dr_treatment_template` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `drug_details`
--

DROP TABLE IF EXISTS `drug_details`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `drug_details` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `ndc` varchar(12) NOT NULL,
  `json` blob NOT NULL,
  `modified_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ndc` (`ndc`)
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
) ENGINE=InnoDB AUTO_INCREMENT=21 DEFAULT CHARSET=utf8;
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
) ENGINE=InnoDB AUTO_INCREMENT=85 DEFAULT CHARSET=utf8;
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
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=utf8;
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
-- Table structure for table `erx_status_events`
--

DROP TABLE IF EXISTS `erx_status_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `erx_status_events` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `treatment_id` int(10) unsigned NOT NULL,
  `erx_status` varchar(100) NOT NULL,
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `status` varchar(100) NOT NULL,
  `event_details` varchar(500) DEFAULT NULL,
  `reported_timestamp` timestamp(6) NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `treatment_id` (`treatment_id`),
  CONSTRAINT `erx_status_events_ibfk_1` FOREIGN KEY (`treatment_id`) REFERENCES `treatment` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `extra_question_fields`
--

DROP TABLE IF EXISTS `extra_question_fields`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `extra_question_fields` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `question_id` int(10) unsigned NOT NULL,
  `json` blob NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `question_id` (`question_id`),
  CONSTRAINT `extra_question_fields_ibfk_1` FOREIGN KEY (`question_id`) REFERENCES `question` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
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
  `medicine_branch` varchar(300) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `treatment_tag` (`health_condition_tag`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `health_log`
--

DROP TABLE IF EXISTS `health_log`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `health_log` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `patient_id` int(10) unsigned NOT NULL,
  `uid` varchar(128) NOT NULL,
  `tstamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `type` varchar(64) NOT NULL,
  `data` blob NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `patient_id` (`patient_id`,`uid`),
  CONSTRAINT `health_log_ibfk_1` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `info_intake`
--

DROP TABLE IF EXISTS `info_intake`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `info_intake` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `context_id` int(10) unsigned NOT NULL,
  `question_id` int(10) unsigned NOT NULL,
  `potential_answer_id` int(10) unsigned DEFAULT NULL,
  `answer_text` varchar(600) DEFAULT NULL,
  `layout_version_id` int(10) unsigned NOT NULL,
  `answered_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `status` varchar(100) NOT NULL,
  `role_id` int(10) unsigned NOT NULL,
  `object_storage_id` int(10) unsigned DEFAULT NULL,
  `parent_info_intake_id` int(10) unsigned DEFAULT NULL,
  `summary_localized_text_id` int(10) unsigned DEFAULT NULL,
  `parent_question_id` int(10) unsigned DEFAULT NULL,
  `role` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_visit_id` (`context_id`),
  KEY `question_id` (`question_id`),
  KEY `potential_answer_id` (`potential_answer_id`),
  KEY `layout_version_id` (`layout_version_id`),
  KEY `patient_id` (`role_id`),
  KEY `object_storage_id` (`object_storage_id`),
  KEY `parent_info_intake_id` (`parent_info_intake_id`),
  KEY `summary_localized_text_id` (`summary_localized_text_id`),
  KEY `parent_question_id` (`parent_question_id`),
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
-- Table structure for table `layout_blob_storage`
--

DROP TABLE IF EXISTS `layout_blob_storage`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `layout_blob_storage` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `layout` blob NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=51 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `layout_version`
--

DROP TABLE IF EXISTS `layout_version`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `layout_version` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `object_storage_id` int(10) unsigned DEFAULT NULL,
  `syntax_version` int(10) unsigned NOT NULL,
  `health_condition_id` int(10) unsigned NOT NULL,
  `comment` varchar(600) DEFAULT NULL,
  `status` varchar(250) NOT NULL,
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `modified_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `role` varchar(250) DEFAULT NULL,
  `layout_purpose` varchar(250) DEFAULT NULL,
  `layout_blob_storage_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `object_storage_id` (`object_storage_id`,`syntax_version`,`health_condition_id`,`status`),
  KEY `treatment_id` (`health_condition_id`),
  KEY `layout_blob_storage_id` (`layout_blob_storage_id`),
  CONSTRAINT `layout_version_ibfk_1` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`),
  CONSTRAINT `layout_version_ibfk_2` FOREIGN KEY (`object_storage_id`) REFERENCES `object_storage` (`id`),
  CONSTRAINT `layout_version_ibfk_3` FOREIGN KEY (`layout_blob_storage_id`) REFERENCES `layout_blob_storage` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=221 DEFAULT CHARSET=utf8;
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
) ENGINE=InnoDB AUTO_INCREMENT=428 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `migrations`
--

DROP TABLE IF EXISTS `migrations`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `migrations` (
  `migration_id` int(10) unsigned NOT NULL,
  `migration_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `migration_user` varchar(100) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `notification_prompt_status`
--

DROP TABLE IF EXISTS `notification_prompt_status`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `notification_prompt_status` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `account_id` int(10) unsigned NOT NULL,
  `prompt_status` varchar(100) NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `account_id` (`account_id`),
  CONSTRAINT `notification_prompt_status_ibfk_1` FOREIGN KEY (`account_id`) REFERENCES `account` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
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
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `modified_date` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00' ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `region_id` (`region_id`,`storage_key`,`bucket`,`status`),
  CONSTRAINT `object_storage_ibfk_1` FOREIGN KEY (`region_id`) REFERENCES `region` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=826 DEFAULT CHARSET=utf8;
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
  `gender` varchar(500) NOT NULL,
  `status` varchar(500) NOT NULL,
  `account_id` int(10) unsigned NOT NULL,
  `erx_patient_id` int(10) unsigned DEFAULT NULL,
  `prefix` varchar(100) DEFAULT NULL,
  `middle_name` varchar(100) DEFAULT NULL,
  `suffix` varchar(100) DEFAULT NULL,
  `payment_service_customer_id` varchar(200) DEFAULT NULL,
  `dob_month` int(10) unsigned DEFAULT NULL,
  `dob_year` int(10) unsigned DEFAULT NULL,
  `dob_day` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `account_id` (`account_id`),
  CONSTRAINT `patient_ibfk_1` FOREIGN KEY (`account_id`) REFERENCES `account` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=91 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_address_selection`
--

DROP TABLE IF EXISTS `patient_address_selection`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_address_selection` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_id` int(10) unsigned NOT NULL,
  `address_id` int(10) unsigned NOT NULL,
  `label` varchar(100) DEFAULT NULL,
  `is_default` tinyint(1) NOT NULL,
  `is_updated_by_doctor` tinyint(1) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_id` (`patient_id`),
  KEY `address_id` (`address_id`),
  CONSTRAINT `patient_address_selection_ibfk_1` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`),
  CONSTRAINT `patient_address_selection_ibfk_2` FOREIGN KEY (`address_id`) REFERENCES `address` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_agreement`
--

DROP TABLE IF EXISTS `patient_agreement`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_agreement` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_id` int(10) unsigned NOT NULL,
  `agreement_type` varchar(100) NOT NULL,
  `status` varchar(100) NOT NULL,
  `agreement_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `agreed` tinyint(1) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_id` (`patient_id`),
  CONSTRAINT `patient_agreement_ibfk_1` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`)
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
  `role_type_id` int(10) unsigned NOT NULL,
  `provider_id` int(10) unsigned NOT NULL,
  `status` varchar(250) NOT NULL,
  `patient_id` int(10) unsigned NOT NULL,
  `expires` timestamp(6) NULL DEFAULT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `health_condition_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `provider_role_id` (`role_type_id`),
  KEY `patient_id` (`patient_id`),
  KEY `health_condition_id` (`health_condition_id`),
  CONSTRAINT `patient_care_provider_assignment_ibfk_5` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`),
  CONSTRAINT `patient_care_provider_assignment_ibfk_3` FOREIGN KEY (`role_type_id`) REFERENCES `role_type` (`id`),
  CONSTRAINT `patient_care_provider_assignment_ibfk_4` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_case`
--

DROP TABLE IF EXISTS `patient_case`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_case` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_id` int(10) unsigned NOT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `status` varchar(100) NOT NULL,
  `health_condition_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_id` (`patient_id`),
  KEY `health_condition_id` (`health_condition_id`),
  CONSTRAINT `patient_case_ibfk_1` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`),
  CONSTRAINT `patient_case_ibfk_2` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_case_care_provider_assignment`
--

DROP TABLE IF EXISTS `patient_case_care_provider_assignment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_case_care_provider_assignment` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_case_id` int(10) unsigned NOT NULL,
  `provider_id` int(10) unsigned NOT NULL,
  `role_type_id` int(10) unsigned NOT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `status` varchar(100) NOT NULL,
  `expires` timestamp(6) NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `patient_case_id` (`patient_case_id`,`provider_id`,`role_type_id`),
  KEY `role_type_id` (`role_type_id`),
  CONSTRAINT `patient_case_care_provider_assignment_ibfk_1` FOREIGN KEY (`patient_case_id`) REFERENCES `patient_case` (`id`),
  CONSTRAINT `patient_case_care_provider_assignment_ibfk_2` FOREIGN KEY (`role_type_id`) REFERENCES `role_type` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_case_message`
--

DROP TABLE IF EXISTS `patient_case_message`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_case_message` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `tstamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `person_id` bigint(20) unsigned NOT NULL,
  `body` text NOT NULL,
  `patient_case_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `person_id` (`person_id`),
  KEY `case_tstamp` (`patient_case_id`,`tstamp`),
  CONSTRAINT `patient_case_message_ibfk_2` FOREIGN KEY (`person_id`) REFERENCES `person` (`id`),
  CONSTRAINT `patient_case_message_ibfk_3` FOREIGN KEY (`patient_case_id`) REFERENCES `patient_case` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_case_message_attachment`
--

DROP TABLE IF EXISTS `patient_case_message_attachment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_case_message_attachment` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `message_id` bigint(20) unsigned DEFAULT NULL,
  `item_type` varchar(64) NOT NULL,
  `item_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `message_id` (`message_id`),
  CONSTRAINT `patient_case_message_attachment_ibfk_1` FOREIGN KEY (`message_id`) REFERENCES `patient_case_message` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `patient_case_message_participant`
--

DROP TABLE IF EXISTS `patient_case_message_participant`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_case_message_participant` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `patient_case_id` int(10) unsigned NOT NULL,
  `person_id` bigint(20) unsigned NOT NULL,
  `unread` tinyint(1) NOT NULL DEFAULT '0',
  `last_read_tstamp` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `patient_case_id` (`patient_case_id`,`person_id`),
  KEY `person_id` (`person_id`),
  CONSTRAINT `patient_case_message_participant_ibfk_1` FOREIGN KEY (`patient_case_id`) REFERENCES `patient_case` (`id`),
  CONSTRAINT `patient_case_message_participant_ibfk_2` FOREIGN KEY (`person_id`) REFERENCES `person` (`id`)
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
  `diagnosis_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
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
  `object_storage_id` int(10) unsigned DEFAULT NULL,
  `language_id` int(10) unsigned NOT NULL,
  `layout_version_id` int(10) unsigned NOT NULL,
  `status` varchar(250) NOT NULL,
  `health_condition_id` int(10) unsigned NOT NULL,
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `modified_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `layout_blob_storage_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `layout_version_id` (`layout_version_id`),
  KEY `language_id` (`language_id`),
  KEY `object_storage_id` (`object_storage_id`),
  KEY `treatment_id` (`health_condition_id`),
  KEY `layout_blob_storage_id` (`layout_blob_storage_id`),
  CONSTRAINT `patient_layout_version_ibfk_1` FOREIGN KEY (`layout_version_id`) REFERENCES `layout_version` (`id`),
  CONSTRAINT `patient_layout_version_ibfk_2` FOREIGN KEY (`language_id`) REFERENCES `languages_supported` (`id`),
  CONSTRAINT `patient_layout_version_ibfk_4` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`),
  CONSTRAINT `patient_layout_version_ibfk_5` FOREIGN KEY (`object_storage_id`) REFERENCES `object_storage` (`id`),
  CONSTRAINT `patient_layout_version_ibfk_6` FOREIGN KEY (`layout_blob_storage_id`) REFERENCES `layout_blob_storage` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=150 DEFAULT CHARSET=utf8;
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
-- Table structure for table `patient_pharmacy_selection`
--

DROP TABLE IF EXISTS `patient_pharmacy_selection`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_pharmacy_selection` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_id` int(10) unsigned NOT NULL,
  `pharmacy_id` varchar(300) DEFAULT NULL,
  `status` varchar(100) NOT NULL,
  `erx_pharmacy_id` int(10) unsigned DEFAULT NULL,
  `pharmacy_selection_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_id` (`patient_id`),
  KEY `pharmacy_selection_id` (`pharmacy_selection_id`),
  CONSTRAINT `patient_pharmacy_selection_ibfk_1` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`),
  CONSTRAINT `patient_pharmacy_selection_ibfk_2` FOREIGN KEY (`pharmacy_selection_id`) REFERENCES `pharmacy_selection` (`id`)
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
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `closed_date` timestamp NULL DEFAULT NULL,
  `health_condition_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  `layout_version_id` int(10) unsigned NOT NULL,
  `submitted_date` timestamp NULL DEFAULT NULL,
  `patient_case_id` int(10) unsigned DEFAULT NULL,
  `diagnosis` text,
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
-- Table structure for table `patient_visit_event`
--

DROP TABLE IF EXISTS `patient_visit_event`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `patient_visit_event` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `patient_visit_id` int(10) unsigned DEFAULT NULL,
  `event` varchar(100) NOT NULL,
  `status` varchar(100) NOT NULL,
  `message` varchar(600) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `patient_visit_id` (`patient_visit_id`),
  CONSTRAINT `patient_visit_event_ibfk_1` FOREIGN KEY (`patient_visit_id`) REFERENCES `patient_visit` (`id`)
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
  `doctor_id` int(10) unsigned NOT NULL,
  `follow_up_date` date NOT NULL,
  `follow_up_value` int(10) unsigned NOT NULL,
  `follow_up_unit` varchar(100) NOT NULL,
  `status` varchar(100) NOT NULL,
  `treatment_plan_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `doctor_id` (`doctor_id`),
  KEY `treatment_plan_id` (`treatment_plan_id`),
  CONSTRAINT `patient_visit_follow_up_ibfk_3` FOREIGN KEY (`treatment_plan_id`) REFERENCES `treatment_plan` (`id`) ON DELETE CASCADE,
  CONSTRAINT `patient_visit_follow_up_ibfk_2` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `pending_task`
--

DROP TABLE IF EXISTS `pending_task`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `pending_task` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `type` varchar(100) NOT NULL,
  `item_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `person`
--

DROP TABLE IF EXISTS `person`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `person` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `role_id` int(10) unsigned NOT NULL,
  `role_type_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `role_type` (`role_id`),
  KEY `role_type_id` (`role_type_id`),
  CONSTRAINT `person_ibfk_1` FOREIGN KEY (`role_type_id`) REFERENCES `role_type` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `pharmacy_dispensed_treatment`
--

DROP TABLE IF EXISTS `pharmacy_dispensed_treatment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `pharmacy_dispensed_treatment` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `drug_internal_name` varchar(250) NOT NULL,
  `dispense_value` decimal(21,10) NOT NULL,
  `refills` int(10) unsigned NOT NULL,
  `substitutions_allowed` tinyint(1) NOT NULL,
  `days_supply` int(10) unsigned DEFAULT NULL,
  `pharmacy_notes` varchar(150) NOT NULL,
  `pharmacy_id` int(10) unsigned NOT NULL,
  `patient_instructions` varchar(150) NOT NULL,
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `status` varchar(100) NOT NULL,
  `dosage_strength` varchar(250) NOT NULL,
  `type` varchar(150) NOT NULL,
  `drug_name_id` int(10) unsigned DEFAULT NULL,
  `drug_form_id` int(10) unsigned DEFAULT NULL,
  `drug_route_id` int(10) unsigned DEFAULT NULL,
  `erx_id` int(10) unsigned NOT NULL,
  `erx_last_filled_date` timestamp NULL DEFAULT NULL,
  `erx_sent_date` timestamp NULL DEFAULT NULL,
  `dispense_unit` varchar(100) NOT NULL,
  `requested_treatment_id` int(10) unsigned DEFAULT NULL,
  `doctor_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `drug_name_id` (`drug_name_id`),
  KEY `drug_route_id` (`drug_route_id`),
  KEY `drug_form_id` (`drug_form_id`),
  KEY `pharmacy_id` (`pharmacy_id`),
  KEY `unlinked_requested_treatment_id` (`requested_treatment_id`),
  KEY `doctor_id` (`doctor_id`),
  CONSTRAINT `pharmacy_dispensed_treatment_ibfk_2` FOREIGN KEY (`drug_name_id`) REFERENCES `drug_name` (`id`),
  CONSTRAINT `pharmacy_dispensed_treatment_ibfk_3` FOREIGN KEY (`drug_route_id`) REFERENCES `drug_route` (`id`),
  CONSTRAINT `pharmacy_dispensed_treatment_ibfk_4` FOREIGN KEY (`drug_form_id`) REFERENCES `drug_form` (`id`),
  CONSTRAINT `pharmacy_dispensed_treatment_ibfk_6` FOREIGN KEY (`pharmacy_id`) REFERENCES `pharmacy_selection` (`id`),
  CONSTRAINT `pharmacy_dispensed_treatment_ibfk_7` FOREIGN KEY (`requested_treatment_id`) REFERENCES `requested_treatment` (`id`),
  CONSTRAINT `pharmacy_dispensed_treatment_ibfk_8` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `pharmacy_dispensed_treatment_drug_db_id`
--

DROP TABLE IF EXISTS `pharmacy_dispensed_treatment_drug_db_id`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `pharmacy_dispensed_treatment_drug_db_id` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `drug_db_id` varchar(100) NOT NULL,
  `drug_db_id_tag` varchar(100) NOT NULL,
  `pharmacy_dispensed_treatment_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `pharmacy_dispensed_treatment_id` (`pharmacy_dispensed_treatment_id`),
  CONSTRAINT `pharmacy_dispensed_treatment_drug_db_id_ibfk_1` FOREIGN KEY (`pharmacy_dispensed_treatment_id`) REFERENCES `pharmacy_dispensed_treatment` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `pharmacy_selection`
--

DROP TABLE IF EXISTS `pharmacy_selection`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `pharmacy_selection` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `pharmacy_id` int(10) unsigned NOT NULL,
  `address_line_1` varchar(500) DEFAULT NULL,
  `address_line_2` varchar(500) DEFAULT NULL,
  `source` varchar(100) NOT NULL,
  `city` varchar(100) DEFAULT NULL,
  `state` varchar(100) DEFAULT NULL,
  `country` varchar(100) DEFAULT NULL,
  `phone` varchar(100) DEFAULT NULL,
  `zip_code` varchar(100) DEFAULT NULL,
  `lat` varchar(100) DEFAULT NULL,
  `lng` varchar(100) DEFAULT NULL,
  `name` varchar(500) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `photo`
--

DROP TABLE IF EXISTS `photo`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `photo` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `uploaded` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `uploader_id` bigint(20) unsigned NOT NULL,
  `mimetype` varchar(128) NOT NULL,
  `url` varchar(255) NOT NULL,
  `claimer_type` varchar(64) DEFAULT NULL,
  `claimer_id` bigint(20) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `uploader_id` (`uploader_id`),
  CONSTRAINT `photo_ibfk_1` FOREIGN KEY (`uploader_id`) REFERENCES `person` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `photo_intake_section`
--

DROP TABLE IF EXISTS `photo_intake_section`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `photo_intake_section` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `section_name` varchar(150) NOT NULL,
  `question_id` int(10) unsigned NOT NULL,
  `patient_id` int(10) unsigned NOT NULL,
  `patient_visit_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `question_id` (`question_id`),
  KEY `patient_id` (`patient_id`),
  KEY `patient_visit_id` (`patient_visit_id`),
  CONSTRAINT `photo_intake_section_ibfk_1` FOREIGN KEY (`question_id`) REFERENCES `question` (`id`),
  CONSTRAINT `photo_intake_section_ibfk_2` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`),
  CONSTRAINT `photo_intake_section_ibfk_3` FOREIGN KEY (`patient_visit_id`) REFERENCES `patient_visit` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `photo_intake_slot`
--

DROP TABLE IF EXISTS `photo_intake_slot`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `photo_intake_slot` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `photo_slot_id` int(10) unsigned NOT NULL,
  `photo_id` int(10) unsigned NOT NULL,
  `photo_slot_name` varchar(150) NOT NULL,
  `photo_intake_section_id` int(10) unsigned NOT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `photo_slot_id` (`photo_slot_id`),
  KEY `photo_id` (`photo_id`),
  KEY `photo_intake_section_id` (`photo_intake_section_id`),
  CONSTRAINT `photo_intake_slot_ibfk_1` FOREIGN KEY (`photo_slot_id`) REFERENCES `photo_slot` (`id`),
  CONSTRAINT `photo_intake_slot_ibfk_2` FOREIGN KEY (`photo_id`) REFERENCES `photo` (`id`),
  CONSTRAINT `photo_intake_slot_ibfk_3` FOREIGN KEY (`photo_intake_section_id`) REFERENCES `photo_intake_section` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `photo_slot`
--

DROP TABLE IF EXISTS `photo_slot`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `photo_slot` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `question_id` int(10) unsigned NOT NULL,
  `slot_name_app_text_id` int(10) unsigned NOT NULL,
  `slot_type_id` int(10) unsigned NOT NULL,
  `required` tinyint(1) NOT NULL,
  `status` varchar(100) NOT NULL,
  `placeholder_image_tag` varchar(100) DEFAULT NULL,
  `ordering` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `question_id` (`question_id`),
  KEY `slot_name_app_text_id` (`slot_name_app_text_id`),
  KEY `slot_type_id` (`slot_type_id`),
  CONSTRAINT `photo_slot_ibfk_1` FOREIGN KEY (`question_id`) REFERENCES `question` (`id`),
  CONSTRAINT `photo_slot_ibfk_2` FOREIGN KEY (`slot_name_app_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `photo_slot_ibfk_3` FOREIGN KEY (`slot_type_id`) REFERENCES `photo_slot_type` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
  `to_alert` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `potential_outcome_tag` (`potential_answer_tag`),
  UNIQUE KEY `question_id_2` (`question_id`,`ordering`),
  KEY `otype_id` (`atype_id`),
  KEY `outcome_localized_text` (`answer_localized_text_id`),
  KEY `answer_summary_text_id` (`answer_summary_text_id`),
  CONSTRAINT `potential_answer_ibfk_1` FOREIGN KEY (`atype_id`) REFERENCES `answer_type` (`id`),
  CONSTRAINT `potential_answer_ibfk_2` FOREIGN KEY (`question_id`) REFERENCES `question` (`id`),
  CONSTRAINT `potential_answer_ibfk_3` FOREIGN KEY (`answer_summary_text_id`) REFERENCES `app_text` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=207 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `push_config`
--

DROP TABLE IF EXISTS `push_config`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `push_config` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `account_id` int(10) unsigned NOT NULL,
  `device_token` varbinary(500) NOT NULL,
  `push_endpoint` varchar(300) NOT NULL,
  `platform` varchar(100) NOT NULL,
  `platform_version` varchar(100) NOT NULL,
  `app_type` varchar(100) NOT NULL,
  `app_env` varchar(100) NOT NULL,
  `app_version` varchar(100) NOT NULL,
  `device` varchar(100) NOT NULL,
  `device_model` varchar(100) NOT NULL,
  `device_id` varchar(100) NOT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `device_token` (`device_token`),
  KEY `account_id` (`account_id`),
  CONSTRAINT `push_config_ibfk_1` FOREIGN KEY (`account_id`) REFERENCES `account` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
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
  `formatted_field_tags` varchar(150) DEFAULT NULL,
  `to_alert` tinyint(1) DEFAULT NULL,
  `alert_app_text_id` int(10) unsigned DEFAULT NULL,
  `qtext_has_tokens` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `question_tag` (`question_tag`),
  KEY `qtype_id` (`qtype_id`),
  KEY `subtext_app_text_id` (`subtext_app_text_id`),
  KEY `qtext_app_text_id` (`qtext_app_text_id`),
  KEY `qtext_short_text_id` (`qtext_short_text_id`),
  KEY `parent_question_id` (`parent_question_id`),
  KEY `alert_app_text_id` (`alert_app_text_id`),
  CONSTRAINT `question_ibfk_1` FOREIGN KEY (`qtype_id`) REFERENCES `question_type` (`id`),
  CONSTRAINT `question_ibfk_2` FOREIGN KEY (`subtext_app_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `question_ibfk_3` FOREIGN KEY (`qtext_app_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `question_ibfk_4` FOREIGN KEY (`qtext_short_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `question_ibfk_5` FOREIGN KEY (`parent_question_id`) REFERENCES `question` (`id`),
  CONSTRAINT `question_ibfk_6` FOREIGN KEY (`alert_app_text_id`) REFERENCES `app_text` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=75 DEFAULT CHARSET=utf8;
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
) ENGINE=InnoDB AUTO_INCREMENT=80 DEFAULT CHARSET=utf8;
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
  `dr_regimen_step_id` int(10) unsigned DEFAULT NULL,
  `status` varchar(100) NOT NULL,
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `treatment_plan_id` int(10) unsigned NOT NULL,
  `text` varchar(150) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `dr_regimen_step_id` (`dr_regimen_step_id`),
  KEY `treatment_plan_id` (`treatment_plan_id`),
  CONSTRAINT `regimen_ibfk_2` FOREIGN KEY (`dr_regimen_step_id`) REFERENCES `dr_regimen_step` (`id`),
  CONSTRAINT `regimen_ibfk_3` FOREIGN KEY (`treatment_plan_id`) REFERENCES `treatment_plan` (`id`) ON DELETE CASCADE
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
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
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
-- Table structure for table `requested_treatment`
--

DROP TABLE IF EXISTS `requested_treatment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `requested_treatment` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `drug_internal_name` varchar(250) NOT NULL,
  `dispense_value` decimal(21,10) NOT NULL,
  `refills` int(10) unsigned NOT NULL,
  `substitutions_allowed` tinyint(1) NOT NULL,
  `days_supply` int(10) unsigned DEFAULT NULL,
  `pharmacy_id` int(10) unsigned NOT NULL,
  `pharmacy_notes` varchar(150) NOT NULL,
  `patient_instructions` varchar(150) NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `status` varchar(100) NOT NULL,
  `dosage_strength` varchar(250) NOT NULL,
  `type` varchar(150) NOT NULL,
  `drug_name_id` int(10) unsigned DEFAULT NULL,
  `drug_form_id` int(10) unsigned DEFAULT NULL,
  `drug_route_id` int(10) unsigned DEFAULT NULL,
  `erx_id` int(10) unsigned DEFAULT NULL,
  `erx_last_filled_date` timestamp NULL DEFAULT NULL,
  `erx_sent_date` timestamp NULL DEFAULT NULL,
  `dispense_unit` varchar(100) NOT NULL,
  `originating_treatment_id` int(10) unsigned DEFAULT NULL,
  `doctor_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `drug_name_id` (`drug_name_id`),
  KEY `drug_route_id` (`drug_route_id`),
  KEY `drug_form_id` (`drug_form_id`),
  KEY `pharmacy_id` (`pharmacy_id`),
  KEY `originating_treatment_id` (`originating_treatment_id`),
  KEY `doctor_id` (`doctor_id`),
  CONSTRAINT `requested_treatment_ibfk_1` FOREIGN KEY (`drug_name_id`) REFERENCES `drug_name` (`id`),
  CONSTRAINT `requested_treatment_ibfk_2` FOREIGN KEY (`drug_route_id`) REFERENCES `drug_route` (`id`),
  CONSTRAINT `requested_treatment_ibfk_3` FOREIGN KEY (`drug_form_id`) REFERENCES `drug_form` (`id`),
  CONSTRAINT `requested_treatment_ibfk_5` FOREIGN KEY (`pharmacy_id`) REFERENCES `pharmacy_selection` (`id`),
  CONSTRAINT `requested_treatment_ibfk_6` FOREIGN KEY (`originating_treatment_id`) REFERENCES `treatment` (`id`),
  CONSTRAINT `requested_treatment_ibfk_7` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `requested_treatment_drug_db_id`
--

DROP TABLE IF EXISTS `requested_treatment_drug_db_id`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `requested_treatment_drug_db_id` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `drug_db_id` int(10) unsigned NOT NULL,
  `drug_db_id_tag` varchar(100) NOT NULL,
  `requested_treatment_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `unlinked_requested_treatment_id` (`requested_treatment_id`),
  CONSTRAINT `requested_treatment_drug_db_id_ibfk_1` FOREIGN KEY (`requested_treatment_id`) REFERENCES `requested_treatment` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `resource_guide`
--

DROP TABLE IF EXISTS `resource_guide`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `resource_guide` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `section_id` int(10) unsigned NOT NULL,
  `ordinal` int(11) NOT NULL,
  `title` varchar(256) NOT NULL,
  `photo_url` varchar(256) NOT NULL,
  `layout` blob NOT NULL,
  `creation_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `section_id` (`section_id`),
  CONSTRAINT `resource_guide_ibfk_1` FOREIGN KEY (`section_id`) REFERENCES `resource_guide_section` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `resource_guide_section`
--

DROP TABLE IF EXISTS `resource_guide_section`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `resource_guide_section` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `ordinal` int(11) NOT NULL,
  `title` varchar(256) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `rx_refill_request`
--

DROP TABLE IF EXISTS `rx_refill_request`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `rx_refill_request` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `erx_request_queue_item_id` int(10) unsigned DEFAULT NULL,
  `reference_number` varchar(100) DEFAULT NULL,
  `pharmacy_rx_reference_number` varchar(100) DEFAULT NULL,
  `patient_id` int(10) unsigned NOT NULL,
  `request_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `doctor_id` int(10) unsigned NOT NULL,
  `dispensed_treatment_id` int(10) unsigned NOT NULL,
  `requested_treatment_id` int(10) unsigned DEFAULT NULL,
  `erx_id` int(10) unsigned DEFAULT NULL,
  `approved_refill_amount` int(10) unsigned DEFAULT NULL,
  `comments` varchar(500) DEFAULT NULL,
  `denial_reason_id` int(10) unsigned DEFAULT NULL,
  `creation_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `dispensed_treatment_id` (`dispensed_treatment_id`),
  KEY `unlinked_requested_treatment_id` (`requested_treatment_id`),
  KEY `doctor_id` (`doctor_id`),
  KEY `patient_id` (`patient_id`),
  KEY `denial_reason_id` (`denial_reason_id`),
  CONSTRAINT `rx_refill_request_ibfk_2` FOREIGN KEY (`dispensed_treatment_id`) REFERENCES `pharmacy_dispensed_treatment` (`id`),
  CONSTRAINT `rx_refill_request_ibfk_3` FOREIGN KEY (`requested_treatment_id`) REFERENCES `requested_treatment` (`id`),
  CONSTRAINT `rx_refill_request_ibfk_4` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`),
  CONSTRAINT `rx_refill_request_ibfk_5` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`),
  CONSTRAINT `rx_refill_request_ibfk_6` FOREIGN KEY (`denial_reason_id`) REFERENCES `deny_refill_reason` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `rx_refill_status_events`
--

DROP TABLE IF EXISTS `rx_refill_status_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `rx_refill_status_events` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `rx_refill_request_id` int(10) unsigned NOT NULL,
  `rx_refill_status` varchar(100) NOT NULL,
  `status` varchar(100) NOT NULL,
  `rx_refill_status_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `reported_timestamp` timestamp(6) NULL DEFAULT NULL,
  `event_details` varchar(500) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `rx_refill_request_id` (`rx_refill_request_id`),
  KEY `status` (`status`),
  CONSTRAINT `rx_refill_status_events_ibfk_1` FOREIGN KEY (`rx_refill_request_id`) REFERENCES `rx_refill_request` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
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
  CONSTRAINT `section_ibfk_1` FOREIGN KEY (`section_title_app_text_id`) REFERENCES `app_text` (`id`),
  CONSTRAINT `section_ibfk_2` FOREIGN KEY (`health_condition_id`) REFERENCES `health_condition` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

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
-- Table structure for table `temp_auth_token`
--

DROP TABLE IF EXISTS `temp_auth_token`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `temp_auth_token` (
  `token` varchar(128) NOT NULL,
  `expires` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `purpose` varchar(32) NOT NULL,
  `account_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`purpose`,`token`),
  KEY `expires` (`expires`),
  KEY `account_id` (`account_id`),
  CONSTRAINT `temp_auth_token_ibfk_1` FOREIGN KEY (`account_id`) REFERENCES `account` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
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
  `dispense_value` decimal(21,10) NOT NULL,
  `dispense_unit_id` int(10) unsigned NOT NULL,
  `refills` int(10) unsigned NOT NULL,
  `substitutions_allowed` tinyint(1) DEFAULT NULL,
  `days_supply` int(10) unsigned DEFAULT NULL,
  `pharmacy_notes` varchar(150) DEFAULT NULL,
  `patient_instructions` varchar(150) NOT NULL,
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `status` varchar(100) NOT NULL,
  `dosage_strength` varchar(250) NOT NULL,
  `type` varchar(150) NOT NULL,
  `drug_name_id` int(10) unsigned DEFAULT NULL,
  `drug_form_id` int(10) unsigned DEFAULT NULL,
  `drug_route_id` int(10) unsigned DEFAULT NULL,
  `erx_sent_date` timestamp NULL DEFAULT NULL,
  `erx_id` int(10) unsigned DEFAULT NULL,
  `pharmacy_id` int(10) unsigned DEFAULT NULL,
  `erx_last_filled_date` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `treatment_plan_id` (`treatment_plan_id`),
  KEY `dispense_unit_id` (`dispense_unit_id`),
  KEY `drug_name_id` (`drug_name_id`),
  KEY `drug_form_id` (`drug_form_id`),
  KEY `drug_route_id` (`drug_route_id`),
  KEY `pharmacy_id` (`pharmacy_id`),
  CONSTRAINT `treatment_ibfk_9` FOREIGN KEY (`treatment_plan_id`) REFERENCES `treatment_plan` (`id`) ON DELETE CASCADE,
  CONSTRAINT `treatment_ibfk_3` FOREIGN KEY (`dispense_unit_id`) REFERENCES `dispense_unit` (`id`),
  CONSTRAINT `treatment_ibfk_5` FOREIGN KEY (`drug_name_id`) REFERENCES `drug_name` (`id`),
  CONSTRAINT `treatment_ibfk_6` FOREIGN KEY (`drug_form_id`) REFERENCES `drug_form` (`id`),
  CONSTRAINT `treatment_ibfk_7` FOREIGN KEY (`drug_route_id`) REFERENCES `drug_route` (`id`),
  CONSTRAINT `treatment_ibfk_8` FOREIGN KEY (`pharmacy_id`) REFERENCES `pharmacy_selection` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `treatment_dr_template_selection`
--

DROP TABLE IF EXISTS `treatment_dr_template_selection`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `treatment_dr_template_selection` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `treatment_id` int(10) unsigned NOT NULL,
  `dr_treatment_template_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `dr_favorite_treatment_id` (`dr_treatment_template_id`),
  KEY `treatment_id` (`treatment_id`),
  CONSTRAINT `treatment_dr_template_selection_ibfk_2` FOREIGN KEY (`treatment_id`) REFERENCES `treatment` (`id`) ON DELETE CASCADE,
  CONSTRAINT `treatment_dr_template_selection_ibfk_1` FOREIGN KEY (`dr_treatment_template_id`) REFERENCES `dr_treatment_template` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `treatment_drug_db_id`
--

DROP TABLE IF EXISTS `treatment_drug_db_id`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `treatment_drug_db_id` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `drug_db_id_tag` varchar(100) NOT NULL,
  `drug_db_id` varchar(100) DEFAULT NULL,
  `treatment_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `treatment_id` (`treatment_id`),
  CONSTRAINT `treatment_drug_db_id_ibfk_1` FOREIGN KEY (`treatment_id`) REFERENCES `treatment` (`id`) ON DELETE CASCADE
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
  CONSTRAINT `treatment_instructions_ibfk_3` FOREIGN KEY (`treatment_id`) REFERENCES `treatment` (`id`) ON DELETE CASCADE,
  CONSTRAINT `treatment_instructions_ibfk_2` FOREIGN KEY (`dr_drug_instruction_id`) REFERENCES `dr_drug_supplemental_instruction` (`id`)
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
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `doctor_id` int(10) unsigned DEFAULT NULL,
  `sent_date` timestamp NULL DEFAULT NULL,
  `patient_id` int(10) unsigned NOT NULL,
  `patient_case_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `doctor_id` (`doctor_id`),
  KEY `patient_id` (`patient_id`),
  CONSTRAINT `treatment_plan_ibfk_2` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`),
  CONSTRAINT `treatment_plan_ibfk_3` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `treatment_plan_content_source`
--

DROP TABLE IF EXISTS `treatment_plan_content_source`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `treatment_plan_content_source` (
  `treatment_plan_id` int(10) unsigned NOT NULL,
  `content_source_id` int(10) unsigned NOT NULL,
  `content_source_type` varchar(100) NOT NULL,
  `doctor_id` int(10) unsigned NOT NULL,
  `has_deviated` tinyint(1) NOT NULL DEFAULT '0',
  `deviated_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`treatment_plan_id`),
  KEY `doctor_id` (`doctor_id`),
  CONSTRAINT `treatment_plan_content_source_ibfk_1` FOREIGN KEY (`treatment_plan_id`) REFERENCES `treatment_plan` (`id`) ON DELETE CASCADE,
  CONSTRAINT `treatment_plan_content_source_ibfk_2` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `treatment_plan_parent`
--

DROP TABLE IF EXISTS `treatment_plan_parent`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `treatment_plan_parent` (
  `treatment_plan_id` int(10) unsigned NOT NULL,
  `parent_id` int(10) unsigned NOT NULL,
  `parent_type` varchar(100) NOT NULL,
  PRIMARY KEY (`treatment_plan_id`),
  CONSTRAINT `treatment_plan_parent_ibfk_1` FOREIGN KEY (`treatment_plan_id`) REFERENCES `treatment_plan` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `treatment_plan_patient_visit_mapping`
--

DROP TABLE IF EXISTS `treatment_plan_patient_visit_mapping`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `treatment_plan_patient_visit_mapping` (
  `treatment_plan_id` int(10) unsigned NOT NULL,
  `patient_visit_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`treatment_plan_id`,`patient_visit_id`),
  KEY `patient_visit_id` (`patient_visit_id`),
  CONSTRAINT `treatment_plan_patient_visit_mapping_ibfk_1` FOREIGN KEY (`treatment_plan_id`) REFERENCES `treatment_plan` (`id`) ON DELETE CASCADE,
  CONSTRAINT `treatment_plan_patient_visit_mapping_ibfk_2` FOREIGN KEY (`patient_visit_id`) REFERENCES `patient_visit` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `unclaimed_case_queue`
--

DROP TABLE IF EXISTS `unclaimed_case_queue`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `unclaimed_case_queue` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `care_providing_state_id` int(10) unsigned NOT NULL,
  `event_type` varchar(100) NOT NULL,
  `item_id` int(10) unsigned NOT NULL,
  `patient_case_id` int(10) unsigned NOT NULL,
  `status` varchar(100) NOT NULL,
  `locked` tinyint(1) NOT NULL DEFAULT '0',
  `doctor_id` int(10) unsigned DEFAULT NULL,
  `enqueue_date` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `expires` timestamp(6) NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `patient_case_id` (`patient_case_id`),
  KEY `doctor_id` (`doctor_id`),
  KEY `care_providing_state_id` (`care_providing_state_id`),
  CONSTRAINT `unclaimed_case_queue_ibfk_1` FOREIGN KEY (`care_providing_state_id`) REFERENCES `care_providing_state` (`id`),
  CONSTRAINT `unclaimed_case_queue_ibfk_2` FOREIGN KEY (`patient_case_id`) REFERENCES `patient_case` (`id`),
  CONSTRAINT `unclaimed_case_queue_ibfk_3` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `unlinked_dntf_treatment`
--

DROP TABLE IF EXISTS `unlinked_dntf_treatment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `unlinked_dntf_treatment` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `drug_internal_name` varchar(250) NOT NULL,
  `dispense_value` decimal(21,10) NOT NULL,
  `dispense_unit_id` int(10) unsigned NOT NULL,
  `refills` int(10) unsigned NOT NULL,
  `days_supply` int(10) unsigned DEFAULT NULL,
  `pharmacy_notes` varchar(150) NOT NULL,
  `substitutions_allowed` tinyint(4) DEFAULT NULL,
  `patient_instructions` varchar(150) NOT NULL,
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `status` varchar(100) NOT NULL,
  `dosage_strength` varchar(250) NOT NULL,
  `type` varchar(150) NOT NULL,
  `drug_name_id` int(10) unsigned DEFAULT NULL,
  `drug_form_id` int(10) unsigned DEFAULT NULL,
  `drug_route_id` int(10) unsigned DEFAULT NULL,
  `erx_sent_date` timestamp NULL DEFAULT NULL,
  `erx_id` int(10) unsigned DEFAULT NULL,
  `pharmacy_id` int(10) unsigned DEFAULT NULL,
  `erx_last_filled_date` timestamp(6) NULL DEFAULT NULL,
  `patient_id` int(10) unsigned NOT NULL,
  `doctor_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `dispense_unit_id` (`dispense_unit_id`),
  KEY `drug_name_id` (`drug_name_id`),
  KEY `drug_form_id` (`drug_form_id`),
  KEY `drug_route_id` (`drug_route_id`),
  KEY `pharmacy_id` (`pharmacy_id`),
  KEY `patient_id` (`patient_id`),
  KEY `doctor_id` (`doctor_id`),
  CONSTRAINT `unlinked_dntf_treatment_ibfk_1` FOREIGN KEY (`dispense_unit_id`) REFERENCES `dispense_unit` (`id`),
  CONSTRAINT `unlinked_dntf_treatment_ibfk_2` FOREIGN KEY (`drug_name_id`) REFERENCES `drug_name` (`id`),
  CONSTRAINT `unlinked_dntf_treatment_ibfk_3` FOREIGN KEY (`drug_form_id`) REFERENCES `drug_form` (`id`),
  CONSTRAINT `unlinked_dntf_treatment_ibfk_4` FOREIGN KEY (`drug_route_id`) REFERENCES `drug_route` (`id`),
  CONSTRAINT `unlinked_dntf_treatment_ibfk_5` FOREIGN KEY (`pharmacy_id`) REFERENCES `pharmacy_selection` (`id`),
  CONSTRAINT `unlinked_dntf_treatment_ibfk_6` FOREIGN KEY (`patient_id`) REFERENCES `patient` (`id`),
  CONSTRAINT `unlinked_dntf_treatment_ibfk_7` FOREIGN KEY (`doctor_id`) REFERENCES `doctor` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `unlinked_dntf_treatment_drug_db_id`
--

DROP TABLE IF EXISTS `unlinked_dntf_treatment_drug_db_id`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `unlinked_dntf_treatment_drug_db_id` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `drug_db_id` varchar(100) NOT NULL,
  `drug_db_id_tag` varchar(100) NOT NULL,
  `unlinked_dntf_treatment_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `unlinked_dntf_treatment_id` (`unlinked_dntf_treatment_id`),
  CONSTRAINT `unlinked_dntf_treatment_drug_db_id_ibfk_1` FOREIGN KEY (`unlinked_dntf_treatment_id`) REFERENCES `unlinked_dntf_treatment` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `unlinked_dntf_treatment_status_events`
--

DROP TABLE IF EXISTS `unlinked_dntf_treatment_status_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `unlinked_dntf_treatment_status_events` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `unlinked_dntf_treatment_id` int(10) unsigned NOT NULL,
  `erx_status` varchar(100) NOT NULL,
  `creation_date` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
  `status` varchar(100) NOT NULL,
  `event_details` varchar(500) DEFAULT NULL,
  `reported_timestamp` timestamp(6) NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `unlinked_dntf_treatment_id` (`unlinked_dntf_treatment_id`),
  CONSTRAINT `unlinked_dntf_treatment_status_events_ibfk_1` FOREIGN KEY (`unlinked_dntf_treatment_id`) REFERENCES `unlinked_dntf_treatment` (`id`)
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

-- Dump completed on 2014-07-15 19:09:25
