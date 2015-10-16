-- Note. This file just represents a schema dump of the carefinder database for the data
-- that we have collected thus far. What is going to be easiest to resume the database from a point in time is to
-- restore from a snapshot as there is a geocoded and user generated data (such as qa data) that cannot be re-created 
-- easily.

--
-- PostgreSQL database dump
--

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


--
-- Name: postgis; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA public;


--
-- Name: EXTENSION postgis; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION postgis IS 'PostGIS geometry, geography, and raster spatial types and functions';


SET search_path = public, pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: business_geocode; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE business_geocode (
    npi text NOT NULL,
    lat numeric(10,6),
    lng numeric(10,6),
    score double precision,
    geom geometry(Point,4326),
    geog geography
);


--
-- Name: cities; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE cities (
    geonameid integer NOT NULL,
    name text,
    asciiname text,
    alternatenames text,
    latitude numeric(10,6),
    longitude numeric(10,6),
    feature_class text,
    feature_code text,
    country_code text,
    cc2 text,
    admin1_code text,
    admin2_code text,
    admin3_code text,
    admin4_code text,
    population bigint,
    elevation text,
    dem double precision NOT NULL,
    timezone text,
    last_modified date NOT NULL,
    geom geometry(Point,4326),
    geog geography
);


--
-- Name: doctor_qa; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE doctor_qa (
    npi text NOT NULL,
    yelp_page_correct boolean,
    yelp_page_correct_comments text,
    yelp_address_match boolean,
    image_1_url text,
    image_1_ok boolean,
    image_1_comments text,
    image_2_url text,
    image_2_ok boolean,
    image_2_comments text,
    image_3_url text,
    image_3_ok boolean,
    image_3_comments text,
    alternate_image_url text
);


--
-- Name: namcs; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE namcs (
    code1 text,
    code1_rule_out boolean,
    code1_bucket text,
    code2 text,
    code2_rule_out boolean,
    code2_bucket text,
    code3 text,
    code3_rule_out boolean,
    code3_bucket text,
    state text,
    year integer,
    code text,
    ordinal integer,
    bucket text,
    rule_out boolean
);


--
-- Name: npi; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE npi (
    npi text,
    entity_type_code text,
    replacement_npi text,
    ein text,
    provider_organization_name text,
    provider_last_name text,
    provider_first_name text,
    provider_middle_name text,
    provider_name_prefix_text text,
    provider_name_suffix_text text,
    provider_credential_text text,
    provider_other_organization_name text,
    provider_other_organization_name_type_code text,
    provider_other_last_name text,
    provider_other_first_name text,
    provider_other_middle_name text,
    provider_other_name_prefix_text text,
    provider_other_name_suffix_text text,
    provider_other_credential_text text,
    provider_other_last_name_type_code text,
    provider_first_line_business_mailing_address text,
    provider_second_line_business_mailing_address text,
    provider_business_mailing_address_city_name text,
    provider_business_mailing_address_state_name text,
    provider_business_mailing_address_postal_code text,
    provider_business_mailing_address_country_code text,
    provider_business_mailing_address_telephone_number text,
    provider_business_mailing_address_fax_number text,
    provider_first_line_business_practice_location_address text,
    provider_second_line_business_practice_location_address text,
    provider_business_practice_location_address_city_name text,
    provider_business_practice_location_address_state_name text,
    provider_business_practice_location_address_postal_code text,
    provider_business_practice_location_address_country_code text,
    provider_business_practice_location_address_telephone_number text,
    provider_business_practice_location_address_fax_number text,
    provider_enumeration_date text,
    last_update_date text,
    npi_deactivation_reason_code text,
    npi_deactivation_date text,
    npi_reactivation_date text,
    provider_gender_code text,
    authorized_official_last_name text,
    authorized_official_first_name text,
    authorized_official_middle_name text,
    authorized_official_title_or_position text,
    authorized_official_telephone_number text,
    healthcare_provider_taxonomy_code_1 text,
    provider_license_number_1 text,
    provider_license_number_state_code_1 text,
    healthcare_provider_primary_taxonomy_switch_1 text,
    healthcare_provider_taxonomy_code_2 text,
    provider_license_number_2 text,
    provider_license_number_state_code_2 text,
    healthcare_provider_primary_taxonomy_switch_2 text,
    healthcare_provider_taxonomy_code_3 text,
    provider_license_number_3 text,
    provider_license_number_state_code_3 text,
    healthcare_provider_primary_taxonomy_switch_3 text,
    healthcare_provider_taxonomy_code_4 text,
    provider_license_number_4 text,
    provider_license_number_state_code_4 text,
    healthcare_provider_primary_taxonomy_switch_4 text,
    healthcare_provider_taxonomy_code_5 text,
    provider_license_number_5 text,
    provider_license_number_state_code_5 text,
    healthcare_provider_primary_taxonomy_switch_5 text,
    healthcare_provider_taxonomy_code_6 text,
    provider_license_number_6 text,
    provider_license_number_state_code_6 text,
    healthcare_provider_primary_taxonomy_switch_6 text,
    healthcare_provider_taxonomy_code_7 text,
    provider_license_number_7 text,
    provider_license_number_state_code_7 text,
    healthcare_provider_primary_taxonomy_switch_7 text,
    healthcare_provider_taxonomy_code_8 text,
    provider_license_number_8 text,
    provider_license_number_state_code_8 text,
    healthcare_provider_primary_taxonomy_switch_8 text,
    healthcare_provider_taxonomy_code_9 text,
    provider_license_number_9 text,
    provider_license_number_state_code_9 text,
    healthcare_provider_primary_taxonomy_switch_9 text,
    healthcare_provider_taxonomy_code_10 text,
    provider_license_number_10 text,
    provider_license_number_state_code_10 text,
    healthcare_provider_primary_taxonomy_switch_10 text,
    healthcare_provider_taxonomy_code_11 text,
    provider_license_number_11 text,
    provider_license_number_state_code_11 text,
    healthcare_provider_primary_taxonomy_switch_11 text,
    healthcare_provider_taxonomy_code_12 text,
    provider_license_number_12 text,
    provider_license_number_state_code_12 text,
    healthcare_provider_primary_taxonomy_switch_12 text,
    healthcare_provider_taxonomy_code_13 text,
    provider_license_number_13 text,
    provider_license_number_state_code_13 text,
    healthcare_provider_primary_taxonomy_switch_13 text,
    healthcare_provider_taxonomy_code_14 text,
    provider_license_number_14 text,
    provider_license_number_state_code_14 text,
    healthcare_provider_primary_taxonomy_switch_14 text,
    healthcare_provider_taxonomy_code_15 text,
    provider_license_number_15 text,
    provider_license_number_state_code_15 text,
    healthcare_provider_primary_taxonomy_switch_15 text,
    other_provider_identifier_1 text,
    other_provider_identifier_type_code_1 text,
    other_provider_identifier_state_1 text,
    other_provider_identifier_issuer_1 text,
    other_provider_identifier_2 text,
    other_provider_identifier_type_code_2 text,
    other_provider_identifier_state_2 text,
    other_provider_identifier_issuer_2 text,
    other_provider_identifier_3 text,
    other_provider_identifier_type_code_3 text,
    other_provider_identifier_state_3 text,
    other_provider_identifier_issuer_3 text,
    other_provider_identifier_4 text,
    other_provider_identifier_type_code_4 text,
    other_provider_identifier_state_4 text,
    other_provider_identifier_issuer_4 text,
    other_provider_identifier_5 text,
    other_provider_identifier_type_code_5 text,
    other_provider_identifier_state_5 text,
    other_provider_identifier_issuer_5 text,
    other_provider_identifier_6 text,
    other_provider_identifier_type_code_6 text,
    other_provider_identifier_state_6 text,
    other_provider_identifier_issuer_6 text,
    other_provider_identifier_7 text,
    other_provider_identifier_type_code_7 text,
    other_provider_identifier_state_7 text,
    other_provider_identifier_issuer_7 text,
    other_provider_identifier_8 text,
    other_provider_identifier_type_code_8 text,
    other_provider_identifier_state_8 text,
    other_provider_identifier_issuer_8 text,
    other_provider_identifier_9 text,
    other_provider_identifier_type_code_9 text,
    other_provider_identifier_state_9 text,
    other_provider_identifier_issuer_9 text,
    other_provider_identifier_10 text,
    other_provider_identifier_type_code_10 text,
    other_provider_identifier_state_10 text,
    other_provider_identifier_issuer_10 text,
    other_provider_identifier_11 text,
    other_provider_identifier_type_code_11 text,
    other_provider_identifier_state_11 text,
    other_provider_identifier_issuer_11 text,
    other_provider_identifier_12 text,
    other_provider_identifier_type_code_12 text,
    other_provider_identifier_state_12 text,
    other_provider_identifier_issuer_12 text,
    other_provider_identifier_13 text,
    other_provider_identifier_type_code_13 text,
    other_provider_identifier_state_13 text,
    other_provider_identifier_issuer_13 text,
    other_provider_identifier_14 text,
    other_provider_identifier_type_code_14 text,
    other_provider_identifier_state_14 text,
    other_provider_identifier_issuer_14 text,
    other_provider_identifier_15 text,
    other_provider_identifier_type_code_15 text,
    other_provider_identifier_state_15 text,
    other_provider_identifier_issuer_15 text,
    other_provider_identifier_16 text,
    other_provider_identifier_type_code_16 text,
    other_provider_identifier_state_16 text,
    other_provider_identifier_issuer_16 text,
    other_provider_identifier_17 text,
    other_provider_identifier_type_code_17 text,
    other_provider_identifier_state_17 text,
    other_provider_identifier_issuer_17 text,
    other_provider_identifier_18 text,
    other_provider_identifier_type_code_18 text,
    other_provider_identifier_state_18 text,
    other_provider_identifier_issuer_18 text,
    other_provider_identifier_19 text,
    other_provider_identifier_type_code_19 text,
    other_provider_identifier_state_19 text,
    other_provider_identifier_issuer_19 text,
    other_provider_identifier_20 text,
    other_provider_identifier_type_code_20 text,
    other_provider_identifier_state_20 text,
    other_provider_identifier_issuer_20 text,
    other_provider_identifier_21 text,
    other_provider_identifier_type_code_21 text,
    other_provider_identifier_state_21 text,
    other_provider_identifier_issuer_21 text,
    other_provider_identifier_22 text,
    other_provider_identifier_type_code_22 text,
    other_provider_identifier_state_22 text,
    other_provider_identifier_issuer_22 text,
    other_provider_identifier_23 text,
    other_provider_identifier_type_code_23 text,
    other_provider_identifier_state_23 text,
    other_provider_identifier_issuer_23 text,
    other_provider_identifier_24 text,
    other_provider_identifier_type_code_24 text,
    other_provider_identifier_state_24 text,
    other_provider_identifier_issuer_24 text,
    other_provider_identifier_25 text,
    other_provider_identifier_type_code_25 text,
    other_provider_identifier_state_25 text,
    other_provider_identifier_issuer_25 text,
    other_provider_identifier_26 text,
    other_provider_identifier_type_code_26 text,
    other_provider_identifier_state_26 text,
    other_provider_identifier_issuer_26 text,
    other_provider_identifier_27 text,
    other_provider_identifier_type_code_27 text,
    other_provider_identifier_state_27 text,
    other_provider_identifier_issuer_27 text,
    other_provider_identifier_28 text,
    other_provider_identifier_type_code_28 text,
    other_provider_identifier_state_28 text,
    other_provider_identifier_issuer_28 text,
    other_provider_identifier_29 text,
    other_provider_identifier_type_code_29 text,
    other_provider_identifier_state_29 text,
    other_provider_identifier_issuer_29 text,
    other_provider_identifier_30 text,
    other_provider_identifier_type_code_30 text,
    other_provider_identifier_state_30 text,
    other_provider_identifier_issuer_30 text,
    other_provider_identifier_31 text,
    other_provider_identifier_type_code_31 text,
    other_provider_identifier_state_31 text,
    other_provider_identifier_issuer_31 text,
    other_provider_identifier_32 text,
    other_provider_identifier_type_code_32 text,
    other_provider_identifier_state_32 text,
    other_provider_identifier_issuer_32 text,
    other_provider_identifier_33 text,
    other_provider_identifier_type_code_33 text,
    other_provider_identifier_state_33 text,
    other_provider_identifier_issuer_33 text,
    other_provider_identifier_34 text,
    other_provider_identifier_type_code_34 text,
    other_provider_identifier_state_34 text,
    other_provider_identifier_issuer_34 text,
    other_provider_identifier_35 text,
    other_provider_identifier_type_code_35 text,
    other_provider_identifier_state_35 text,
    other_provider_identifier_issuer_35 text,
    other_provider_identifier_36 text,
    other_provider_identifier_type_code_36 text,
    other_provider_identifier_state_36 text,
    other_provider_identifier_issuer_36 text,
    other_provider_identifier_37 text,
    other_provider_identifier_type_code_37 text,
    other_provider_identifier_state_37 text,
    other_provider_identifier_issuer_37 text,
    other_provider_identifier_38 text,
    other_provider_identifier_type_code_38 text,
    other_provider_identifier_state_38 text,
    other_provider_identifier_issuer_38 text,
    other_provider_identifier_39 text,
    other_provider_identifier_type_code_39 text,
    other_provider_identifier_state_39 text,
    other_provider_identifier_issuer_39 text,
    other_provider_identifier_40 text,
    other_provider_identifier_type_code_40 text,
    other_provider_identifier_state_40 text,
    other_provider_identifier_issuer_40 text,
    other_provider_identifier_41 text,
    other_provider_identifier_type_code_41 text,
    other_provider_identifier_state_41 text,
    other_provider_identifier_issuer_41 text,
    other_provider_identifier_42 text,
    other_provider_identifier_type_code_42 text,
    other_provider_identifier_state_42 text,
    other_provider_identifier_issuer_42 text,
    other_provider_identifier_43 text,
    other_provider_identifier_type_code_43 text,
    other_provider_identifier_state_43 text,
    other_provider_identifier_issuer_43 text,
    other_provider_identifier_44 text,
    other_provider_identifier_type_code_44 text,
    other_provider_identifier_state_44 text,
    other_provider_identifier_issuer_44 text,
    other_provider_identifier_45 text,
    other_provider_identifier_type_code_45 text,
    other_provider_identifier_state_45 text,
    other_provider_identifier_issuer_45 text,
    other_provider_identifier_46 text,
    other_provider_identifier_type_code_46 text,
    other_provider_identifier_state_46 text,
    other_provider_identifier_issuer_46 text,
    other_provider_identifier_47 text,
    other_provider_identifier_type_code_47 text,
    other_provider_identifier_state_47 text,
    other_provider_identifier_issuer_47 text,
    other_provider_identifier_48 text,
    other_provider_identifier_type_code_48 text,
    other_provider_identifier_state_48 text,
    other_provider_identifier_issuer_48 text,
    other_provider_identifier_49 text,
    other_provider_identifier_type_code_49 text,
    other_provider_identifier_state_49 text,
    other_provider_identifier_issuer_49 text,
    other_provider_identifier_50 text,
    other_provider_identifier_type_code_50 text,
    other_provider_identifier_state_50 text,
    other_provider_identifier_issuer_50 text,
    is_sole_proprietor text,
    is_organization_subpart text,
    parent_organization_lbn text,
    parent_organization_tin text,
    authorized_official_name_prefix_text text,
    authorized_official_name_suffix_text text,
    authorized_official_credential_text text,
    healthcare_provider_taxonomy_group_1 text,
    healthcare_provider_taxonomy_group_2 text,
    healthcare_provider_taxonomy_group_3 text,
    healthcare_provider_taxonomy_group_4 text,
    healthcare_provider_taxonomy_group_5 text,
    healthcare_provider_taxonomy_group_6 text,
    healthcare_provider_taxonomy_group_7 text,
    healthcare_provider_taxonomy_group_8 text,
    healthcare_provider_taxonomy_group_9 text,
    healthcare_provider_taxonomy_group_10 text,
    healthcare_provider_taxonomy_group_11 text,
    healthcare_provider_taxonomy_group_12 text,
    healthcare_provider_taxonomy_group_13 text,
    healthcare_provider_taxonomy_group_14 text,
    healthcare_provider_taxonomy_group_15 text,
    better_doctor_json_text_old text DEFAULT to_json('{}'::text),
    bing_images text,
    google_images text,
    better_doctor_json_backup text,
    better_doctor_jsonb jsonb
);


--
-- Name: taxonomy_code; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE taxonomy_code (
    npi text,
    code text,
    is_primary boolean
);


--
-- Name: npi_derms_only; Type: MATERIALIZED VIEW; Schema: public; Owner: -; Tablespace: 
--

CREATE MATERIALIZED VIEW npi_derms_only AS
 WITH derms AS (
         SELECT taxonomy_code.npi AS taxonomy_code_npi
           FROM taxonomy_code
          WHERE ((taxonomy_code.is_primary = true) AND (taxonomy_code.code = '207N00000X'::text))
        )
 SELECT npi.npi,
    npi.entity_type_code,
    npi.replacement_npi,
    npi.ein,
    npi.provider_organization_name,
    npi.provider_last_name,
    npi.provider_first_name,
    npi.provider_middle_name,
    npi.provider_name_prefix_text,
    npi.provider_name_suffix_text,
    npi.provider_credential_text,
    npi.provider_other_organization_name,
    npi.provider_other_organization_name_type_code,
    npi.provider_other_last_name,
    npi.provider_other_first_name,
    npi.provider_other_middle_name,
    npi.provider_other_name_prefix_text,
    npi.provider_other_name_suffix_text,
    npi.provider_other_credential_text,
    npi.provider_other_last_name_type_code,
    npi.provider_first_line_business_mailing_address,
    npi.provider_second_line_business_mailing_address,
    npi.provider_business_mailing_address_city_name,
    npi.provider_business_mailing_address_state_name,
    npi.provider_business_mailing_address_postal_code,
    npi.provider_business_mailing_address_country_code,
    npi.provider_business_mailing_address_telephone_number,
    npi.provider_business_mailing_address_fax_number,
    npi.provider_first_line_business_practice_location_address,
    npi.provider_second_line_business_practice_location_address,
    npi.provider_business_practice_location_address_city_name,
    npi.provider_business_practice_location_address_state_name,
    npi.provider_business_practice_location_address_postal_code,
    npi.provider_business_practice_location_address_country_code,
    npi.provider_business_practice_location_address_telephone_number,
    npi.provider_business_practice_location_address_fax_number,
    npi.provider_enumeration_date,
    npi.last_update_date,
    npi.npi_deactivation_reason_code,
    npi.npi_deactivation_date,
    npi.npi_reactivation_date,
    npi.provider_gender_code,
    npi.authorized_official_last_name,
    npi.authorized_official_first_name,
    npi.authorized_official_middle_name,
    npi.authorized_official_title_or_position,
    npi.authorized_official_telephone_number,
    npi.healthcare_provider_taxonomy_code_1,
    npi.provider_license_number_1,
    npi.provider_license_number_state_code_1,
    npi.healthcare_provider_primary_taxonomy_switch_1,
    npi.healthcare_provider_taxonomy_code_2,
    npi.provider_license_number_2,
    npi.provider_license_number_state_code_2,
    npi.healthcare_provider_primary_taxonomy_switch_2,
    npi.healthcare_provider_taxonomy_code_3,
    npi.provider_license_number_3,
    npi.provider_license_number_state_code_3,
    npi.healthcare_provider_primary_taxonomy_switch_3,
    npi.healthcare_provider_taxonomy_code_4,
    npi.provider_license_number_4,
    npi.provider_license_number_state_code_4,
    npi.healthcare_provider_primary_taxonomy_switch_4,
    npi.healthcare_provider_taxonomy_code_5,
    npi.provider_license_number_5,
    npi.provider_license_number_state_code_5,
    npi.healthcare_provider_primary_taxonomy_switch_5,
    npi.healthcare_provider_taxonomy_code_6,
    npi.provider_license_number_6,
    npi.provider_license_number_state_code_6,
    npi.healthcare_provider_primary_taxonomy_switch_6,
    npi.healthcare_provider_taxonomy_code_7,
    npi.provider_license_number_7,
    npi.provider_license_number_state_code_7,
    npi.healthcare_provider_primary_taxonomy_switch_7,
    npi.healthcare_provider_taxonomy_code_8,
    npi.provider_license_number_8,
    npi.provider_license_number_state_code_8,
    npi.healthcare_provider_primary_taxonomy_switch_8,
    npi.healthcare_provider_taxonomy_code_9,
    npi.provider_license_number_9,
    npi.provider_license_number_state_code_9,
    npi.healthcare_provider_primary_taxonomy_switch_9,
    npi.healthcare_provider_taxonomy_code_10,
    npi.provider_license_number_10,
    npi.provider_license_number_state_code_10,
    npi.healthcare_provider_primary_taxonomy_switch_10,
    npi.healthcare_provider_taxonomy_code_11,
    npi.provider_license_number_11,
    npi.provider_license_number_state_code_11,
    npi.healthcare_provider_primary_taxonomy_switch_11,
    npi.healthcare_provider_taxonomy_code_12,
    npi.provider_license_number_12,
    npi.provider_license_number_state_code_12,
    npi.healthcare_provider_primary_taxonomy_switch_12,
    npi.healthcare_provider_taxonomy_code_13,
    npi.provider_license_number_13,
    npi.provider_license_number_state_code_13,
    npi.healthcare_provider_primary_taxonomy_switch_13,
    npi.healthcare_provider_taxonomy_code_14,
    npi.provider_license_number_14,
    npi.provider_license_number_state_code_14,
    npi.healthcare_provider_primary_taxonomy_switch_14,
    npi.healthcare_provider_taxonomy_code_15,
    npi.provider_license_number_15,
    npi.provider_license_number_state_code_15,
    npi.healthcare_provider_primary_taxonomy_switch_15,
    npi.other_provider_identifier_1,
    npi.other_provider_identifier_type_code_1,
    npi.other_provider_identifier_state_1,
    npi.other_provider_identifier_issuer_1,
    npi.other_provider_identifier_2,
    npi.other_provider_identifier_type_code_2,
    npi.other_provider_identifier_state_2,
    npi.other_provider_identifier_issuer_2,
    npi.other_provider_identifier_3,
    npi.other_provider_identifier_type_code_3,
    npi.other_provider_identifier_state_3,
    npi.other_provider_identifier_issuer_3,
    npi.other_provider_identifier_4,
    npi.other_provider_identifier_type_code_4,
    npi.other_provider_identifier_state_4,
    npi.other_provider_identifier_issuer_4,
    npi.other_provider_identifier_5,
    npi.other_provider_identifier_type_code_5,
    npi.other_provider_identifier_state_5,
    npi.other_provider_identifier_issuer_5,
    npi.other_provider_identifier_6,
    npi.other_provider_identifier_type_code_6,
    npi.other_provider_identifier_state_6,
    npi.other_provider_identifier_issuer_6,
    npi.other_provider_identifier_7,
    npi.other_provider_identifier_type_code_7,
    npi.other_provider_identifier_state_7,
    npi.other_provider_identifier_issuer_7,
    npi.other_provider_identifier_8,
    npi.other_provider_identifier_type_code_8,
    npi.other_provider_identifier_state_8,
    npi.other_provider_identifier_issuer_8,
    npi.other_provider_identifier_9,
    npi.other_provider_identifier_type_code_9,
    npi.other_provider_identifier_state_9,
    npi.other_provider_identifier_issuer_9,
    npi.other_provider_identifier_10,
    npi.other_provider_identifier_type_code_10,
    npi.other_provider_identifier_state_10,
    npi.other_provider_identifier_issuer_10,
    npi.other_provider_identifier_11,
    npi.other_provider_identifier_type_code_11,
    npi.other_provider_identifier_state_11,
    npi.other_provider_identifier_issuer_11,
    npi.other_provider_identifier_12,
    npi.other_provider_identifier_type_code_12,
    npi.other_provider_identifier_state_12,
    npi.other_provider_identifier_issuer_12,
    npi.other_provider_identifier_13,
    npi.other_provider_identifier_type_code_13,
    npi.other_provider_identifier_state_13,
    npi.other_provider_identifier_issuer_13,
    npi.other_provider_identifier_14,
    npi.other_provider_identifier_type_code_14,
    npi.other_provider_identifier_state_14,
    npi.other_provider_identifier_issuer_14,
    npi.other_provider_identifier_15,
    npi.other_provider_identifier_type_code_15,
    npi.other_provider_identifier_state_15,
    npi.other_provider_identifier_issuer_15,
    npi.other_provider_identifier_16,
    npi.other_provider_identifier_type_code_16,
    npi.other_provider_identifier_state_16,
    npi.other_provider_identifier_issuer_16,
    npi.other_provider_identifier_17,
    npi.other_provider_identifier_type_code_17,
    npi.other_provider_identifier_state_17,
    npi.other_provider_identifier_issuer_17,
    npi.other_provider_identifier_18,
    npi.other_provider_identifier_type_code_18,
    npi.other_provider_identifier_state_18,
    npi.other_provider_identifier_issuer_18,
    npi.other_provider_identifier_19,
    npi.other_provider_identifier_type_code_19,
    npi.other_provider_identifier_state_19,
    npi.other_provider_identifier_issuer_19,
    npi.other_provider_identifier_20,
    npi.other_provider_identifier_type_code_20,
    npi.other_provider_identifier_state_20,
    npi.other_provider_identifier_issuer_20,
    npi.other_provider_identifier_21,
    npi.other_provider_identifier_type_code_21,
    npi.other_provider_identifier_state_21,
    npi.other_provider_identifier_issuer_21,
    npi.other_provider_identifier_22,
    npi.other_provider_identifier_type_code_22,
    npi.other_provider_identifier_state_22,
    npi.other_provider_identifier_issuer_22,
    npi.other_provider_identifier_23,
    npi.other_provider_identifier_type_code_23,
    npi.other_provider_identifier_state_23,
    npi.other_provider_identifier_issuer_23,
    npi.other_provider_identifier_24,
    npi.other_provider_identifier_type_code_24,
    npi.other_provider_identifier_state_24,
    npi.other_provider_identifier_issuer_24,
    npi.other_provider_identifier_25,
    npi.other_provider_identifier_type_code_25,
    npi.other_provider_identifier_state_25,
    npi.other_provider_identifier_issuer_25,
    npi.other_provider_identifier_26,
    npi.other_provider_identifier_type_code_26,
    npi.other_provider_identifier_state_26,
    npi.other_provider_identifier_issuer_26,
    npi.other_provider_identifier_27,
    npi.other_provider_identifier_type_code_27,
    npi.other_provider_identifier_state_27,
    npi.other_provider_identifier_issuer_27,
    npi.other_provider_identifier_28,
    npi.other_provider_identifier_type_code_28,
    npi.other_provider_identifier_state_28,
    npi.other_provider_identifier_issuer_28,
    npi.other_provider_identifier_29,
    npi.other_provider_identifier_type_code_29,
    npi.other_provider_identifier_state_29,
    npi.other_provider_identifier_issuer_29,
    npi.other_provider_identifier_30,
    npi.other_provider_identifier_type_code_30,
    npi.other_provider_identifier_state_30,
    npi.other_provider_identifier_issuer_30,
    npi.other_provider_identifier_31,
    npi.other_provider_identifier_type_code_31,
    npi.other_provider_identifier_state_31,
    npi.other_provider_identifier_issuer_31,
    npi.other_provider_identifier_32,
    npi.other_provider_identifier_type_code_32,
    npi.other_provider_identifier_state_32,
    npi.other_provider_identifier_issuer_32,
    npi.other_provider_identifier_33,
    npi.other_provider_identifier_type_code_33,
    npi.other_provider_identifier_state_33,
    npi.other_provider_identifier_issuer_33,
    npi.other_provider_identifier_34,
    npi.other_provider_identifier_type_code_34,
    npi.other_provider_identifier_state_34,
    npi.other_provider_identifier_issuer_34,
    npi.other_provider_identifier_35,
    npi.other_provider_identifier_type_code_35,
    npi.other_provider_identifier_state_35,
    npi.other_provider_identifier_issuer_35,
    npi.other_provider_identifier_36,
    npi.other_provider_identifier_type_code_36,
    npi.other_provider_identifier_state_36,
    npi.other_provider_identifier_issuer_36,
    npi.other_provider_identifier_37,
    npi.other_provider_identifier_type_code_37,
    npi.other_provider_identifier_state_37,
    npi.other_provider_identifier_issuer_37,
    npi.other_provider_identifier_38,
    npi.other_provider_identifier_type_code_38,
    npi.other_provider_identifier_state_38,
    npi.other_provider_identifier_issuer_38,
    npi.other_provider_identifier_39,
    npi.other_provider_identifier_type_code_39,
    npi.other_provider_identifier_state_39,
    npi.other_provider_identifier_issuer_39,
    npi.other_provider_identifier_40,
    npi.other_provider_identifier_type_code_40,
    npi.other_provider_identifier_state_40,
    npi.other_provider_identifier_issuer_40,
    npi.other_provider_identifier_41,
    npi.other_provider_identifier_type_code_41,
    npi.other_provider_identifier_state_41,
    npi.other_provider_identifier_issuer_41,
    npi.other_provider_identifier_42,
    npi.other_provider_identifier_type_code_42,
    npi.other_provider_identifier_state_42,
    npi.other_provider_identifier_issuer_42,
    npi.other_provider_identifier_43,
    npi.other_provider_identifier_type_code_43,
    npi.other_provider_identifier_state_43,
    npi.other_provider_identifier_issuer_43,
    npi.other_provider_identifier_44,
    npi.other_provider_identifier_type_code_44,
    npi.other_provider_identifier_state_44,
    npi.other_provider_identifier_issuer_44,
    npi.other_provider_identifier_45,
    npi.other_provider_identifier_type_code_45,
    npi.other_provider_identifier_state_45,
    npi.other_provider_identifier_issuer_45,
    npi.other_provider_identifier_46,
    npi.other_provider_identifier_type_code_46,
    npi.other_provider_identifier_state_46,
    npi.other_provider_identifier_issuer_46,
    npi.other_provider_identifier_47,
    npi.other_provider_identifier_type_code_47,
    npi.other_provider_identifier_state_47,
    npi.other_provider_identifier_issuer_47,
    npi.other_provider_identifier_48,
    npi.other_provider_identifier_type_code_48,
    npi.other_provider_identifier_state_48,
    npi.other_provider_identifier_issuer_48,
    npi.other_provider_identifier_49,
    npi.other_provider_identifier_type_code_49,
    npi.other_provider_identifier_state_49,
    npi.other_provider_identifier_issuer_49,
    npi.other_provider_identifier_50,
    npi.other_provider_identifier_type_code_50,
    npi.other_provider_identifier_state_50,
    npi.other_provider_identifier_issuer_50,
    npi.is_sole_proprietor,
    npi.is_organization_subpart,
    npi.parent_organization_lbn,
    npi.parent_organization_tin,
    npi.authorized_official_name_prefix_text,
    npi.authorized_official_name_suffix_text,
    npi.authorized_official_credential_text,
    npi.healthcare_provider_taxonomy_group_1,
    npi.healthcare_provider_taxonomy_group_2,
    npi.healthcare_provider_taxonomy_group_3,
    npi.healthcare_provider_taxonomy_group_4,
    npi.healthcare_provider_taxonomy_group_5,
    npi.healthcare_provider_taxonomy_group_6,
    npi.healthcare_provider_taxonomy_group_7,
    npi.healthcare_provider_taxonomy_group_8,
    npi.healthcare_provider_taxonomy_group_9,
    npi.healthcare_provider_taxonomy_group_10,
    npi.healthcare_provider_taxonomy_group_11,
    npi.healthcare_provider_taxonomy_group_12,
    npi.healthcare_provider_taxonomy_group_13,
    npi.healthcare_provider_taxonomy_group_14,
    npi.healthcare_provider_taxonomy_group_15,
    npi.better_doctor_json_text_old,
    npi.bing_images,
    npi.google_images,
    npi.better_doctor_json_backup,
    npi.better_doctor_jsonb,
    derms.taxonomy_code_npi
   FROM (( SELECT npi_1.npi,
            npi_1.entity_type_code,
            npi_1.replacement_npi,
            npi_1.ein,
            npi_1.provider_organization_name,
            npi_1.provider_last_name,
            npi_1.provider_first_name,
            npi_1.provider_middle_name,
            npi_1.provider_name_prefix_text,
            npi_1.provider_name_suffix_text,
            npi_1.provider_credential_text,
            npi_1.provider_other_organization_name,
            npi_1.provider_other_organization_name_type_code,
            npi_1.provider_other_last_name,
            npi_1.provider_other_first_name,
            npi_1.provider_other_middle_name,
            npi_1.provider_other_name_prefix_text,
            npi_1.provider_other_name_suffix_text,
            npi_1.provider_other_credential_text,
            npi_1.provider_other_last_name_type_code,
            npi_1.provider_first_line_business_mailing_address,
            npi_1.provider_second_line_business_mailing_address,
            npi_1.provider_business_mailing_address_city_name,
            npi_1.provider_business_mailing_address_state_name,
            npi_1.provider_business_mailing_address_postal_code,
            npi_1.provider_business_mailing_address_country_code,
            npi_1.provider_business_mailing_address_telephone_number,
            npi_1.provider_business_mailing_address_fax_number,
            npi_1.provider_first_line_business_practice_location_address,
            npi_1.provider_second_line_business_practice_location_address,
            npi_1.provider_business_practice_location_address_city_name,
            npi_1.provider_business_practice_location_address_state_name,
            npi_1.provider_business_practice_location_address_postal_code,
            npi_1.provider_business_practice_location_address_country_code,
            npi_1.provider_business_practice_location_address_telephone_number,
            npi_1.provider_business_practice_location_address_fax_number,
            npi_1.provider_enumeration_date,
            npi_1.last_update_date,
            npi_1.npi_deactivation_reason_code,
            npi_1.npi_deactivation_date,
            npi_1.npi_reactivation_date,
            npi_1.provider_gender_code,
            npi_1.authorized_official_last_name,
            npi_1.authorized_official_first_name,
            npi_1.authorized_official_middle_name,
            npi_1.authorized_official_title_or_position,
            npi_1.authorized_official_telephone_number,
            npi_1.healthcare_provider_taxonomy_code_1,
            npi_1.provider_license_number_1,
            npi_1.provider_license_number_state_code_1,
            npi_1.healthcare_provider_primary_taxonomy_switch_1,
            npi_1.healthcare_provider_taxonomy_code_2,
            npi_1.provider_license_number_2,
            npi_1.provider_license_number_state_code_2,
            npi_1.healthcare_provider_primary_taxonomy_switch_2,
            npi_1.healthcare_provider_taxonomy_code_3,
            npi_1.provider_license_number_3,
            npi_1.provider_license_number_state_code_3,
            npi_1.healthcare_provider_primary_taxonomy_switch_3,
            npi_1.healthcare_provider_taxonomy_code_4,
            npi_1.provider_license_number_4,
            npi_1.provider_license_number_state_code_4,
            npi_1.healthcare_provider_primary_taxonomy_switch_4,
            npi_1.healthcare_provider_taxonomy_code_5,
            npi_1.provider_license_number_5,
            npi_1.provider_license_number_state_code_5,
            npi_1.healthcare_provider_primary_taxonomy_switch_5,
            npi_1.healthcare_provider_taxonomy_code_6,
            npi_1.provider_license_number_6,
            npi_1.provider_license_number_state_code_6,
            npi_1.healthcare_provider_primary_taxonomy_switch_6,
            npi_1.healthcare_provider_taxonomy_code_7,
            npi_1.provider_license_number_7,
            npi_1.provider_license_number_state_code_7,
            npi_1.healthcare_provider_primary_taxonomy_switch_7,
            npi_1.healthcare_provider_taxonomy_code_8,
            npi_1.provider_license_number_8,
            npi_1.provider_license_number_state_code_8,
            npi_1.healthcare_provider_primary_taxonomy_switch_8,
            npi_1.healthcare_provider_taxonomy_code_9,
            npi_1.provider_license_number_9,
            npi_1.provider_license_number_state_code_9,
            npi_1.healthcare_provider_primary_taxonomy_switch_9,
            npi_1.healthcare_provider_taxonomy_code_10,
            npi_1.provider_license_number_10,
            npi_1.provider_license_number_state_code_10,
            npi_1.healthcare_provider_primary_taxonomy_switch_10,
            npi_1.healthcare_provider_taxonomy_code_11,
            npi_1.provider_license_number_11,
            npi_1.provider_license_number_state_code_11,
            npi_1.healthcare_provider_primary_taxonomy_switch_11,
            npi_1.healthcare_provider_taxonomy_code_12,
            npi_1.provider_license_number_12,
            npi_1.provider_license_number_state_code_12,
            npi_1.healthcare_provider_primary_taxonomy_switch_12,
            npi_1.healthcare_provider_taxonomy_code_13,
            npi_1.provider_license_number_13,
            npi_1.provider_license_number_state_code_13,
            npi_1.healthcare_provider_primary_taxonomy_switch_13,
            npi_1.healthcare_provider_taxonomy_code_14,
            npi_1.provider_license_number_14,
            npi_1.provider_license_number_state_code_14,
            npi_1.healthcare_provider_primary_taxonomy_switch_14,
            npi_1.healthcare_provider_taxonomy_code_15,
            npi_1.provider_license_number_15,
            npi_1.provider_license_number_state_code_15,
            npi_1.healthcare_provider_primary_taxonomy_switch_15,
            npi_1.other_provider_identifier_1,
            npi_1.other_provider_identifier_type_code_1,
            npi_1.other_provider_identifier_state_1,
            npi_1.other_provider_identifier_issuer_1,
            npi_1.other_provider_identifier_2,
            npi_1.other_provider_identifier_type_code_2,
            npi_1.other_provider_identifier_state_2,
            npi_1.other_provider_identifier_issuer_2,
            npi_1.other_provider_identifier_3,
            npi_1.other_provider_identifier_type_code_3,
            npi_1.other_provider_identifier_state_3,
            npi_1.other_provider_identifier_issuer_3,
            npi_1.other_provider_identifier_4,
            npi_1.other_provider_identifier_type_code_4,
            npi_1.other_provider_identifier_state_4,
            npi_1.other_provider_identifier_issuer_4,
            npi_1.other_provider_identifier_5,
            npi_1.other_provider_identifier_type_code_5,
            npi_1.other_provider_identifier_state_5,
            npi_1.other_provider_identifier_issuer_5,
            npi_1.other_provider_identifier_6,
            npi_1.other_provider_identifier_type_code_6,
            npi_1.other_provider_identifier_state_6,
            npi_1.other_provider_identifier_issuer_6,
            npi_1.other_provider_identifier_7,
            npi_1.other_provider_identifier_type_code_7,
            npi_1.other_provider_identifier_state_7,
            npi_1.other_provider_identifier_issuer_7,
            npi_1.other_provider_identifier_8,
            npi_1.other_provider_identifier_type_code_8,
            npi_1.other_provider_identifier_state_8,
            npi_1.other_provider_identifier_issuer_8,
            npi_1.other_provider_identifier_9,
            npi_1.other_provider_identifier_type_code_9,
            npi_1.other_provider_identifier_state_9,
            npi_1.other_provider_identifier_issuer_9,
            npi_1.other_provider_identifier_10,
            npi_1.other_provider_identifier_type_code_10,
            npi_1.other_provider_identifier_state_10,
            npi_1.other_provider_identifier_issuer_10,
            npi_1.other_provider_identifier_11,
            npi_1.other_provider_identifier_type_code_11,
            npi_1.other_provider_identifier_state_11,
            npi_1.other_provider_identifier_issuer_11,
            npi_1.other_provider_identifier_12,
            npi_1.other_provider_identifier_type_code_12,
            npi_1.other_provider_identifier_state_12,
            npi_1.other_provider_identifier_issuer_12,
            npi_1.other_provider_identifier_13,
            npi_1.other_provider_identifier_type_code_13,
            npi_1.other_provider_identifier_state_13,
            npi_1.other_provider_identifier_issuer_13,
            npi_1.other_provider_identifier_14,
            npi_1.other_provider_identifier_type_code_14,
            npi_1.other_provider_identifier_state_14,
            npi_1.other_provider_identifier_issuer_14,
            npi_1.other_provider_identifier_15,
            npi_1.other_provider_identifier_type_code_15,
            npi_1.other_provider_identifier_state_15,
            npi_1.other_provider_identifier_issuer_15,
            npi_1.other_provider_identifier_16,
            npi_1.other_provider_identifier_type_code_16,
            npi_1.other_provider_identifier_state_16,
            npi_1.other_provider_identifier_issuer_16,
            npi_1.other_provider_identifier_17,
            npi_1.other_provider_identifier_type_code_17,
            npi_1.other_provider_identifier_state_17,
            npi_1.other_provider_identifier_issuer_17,
            npi_1.other_provider_identifier_18,
            npi_1.other_provider_identifier_type_code_18,
            npi_1.other_provider_identifier_state_18,
            npi_1.other_provider_identifier_issuer_18,
            npi_1.other_provider_identifier_19,
            npi_1.other_provider_identifier_type_code_19,
            npi_1.other_provider_identifier_state_19,
            npi_1.other_provider_identifier_issuer_19,
            npi_1.other_provider_identifier_20,
            npi_1.other_provider_identifier_type_code_20,
            npi_1.other_provider_identifier_state_20,
            npi_1.other_provider_identifier_issuer_20,
            npi_1.other_provider_identifier_21,
            npi_1.other_provider_identifier_type_code_21,
            npi_1.other_provider_identifier_state_21,
            npi_1.other_provider_identifier_issuer_21,
            npi_1.other_provider_identifier_22,
            npi_1.other_provider_identifier_type_code_22,
            npi_1.other_provider_identifier_state_22,
            npi_1.other_provider_identifier_issuer_22,
            npi_1.other_provider_identifier_23,
            npi_1.other_provider_identifier_type_code_23,
            npi_1.other_provider_identifier_state_23,
            npi_1.other_provider_identifier_issuer_23,
            npi_1.other_provider_identifier_24,
            npi_1.other_provider_identifier_type_code_24,
            npi_1.other_provider_identifier_state_24,
            npi_1.other_provider_identifier_issuer_24,
            npi_1.other_provider_identifier_25,
            npi_1.other_provider_identifier_type_code_25,
            npi_1.other_provider_identifier_state_25,
            npi_1.other_provider_identifier_issuer_25,
            npi_1.other_provider_identifier_26,
            npi_1.other_provider_identifier_type_code_26,
            npi_1.other_provider_identifier_state_26,
            npi_1.other_provider_identifier_issuer_26,
            npi_1.other_provider_identifier_27,
            npi_1.other_provider_identifier_type_code_27,
            npi_1.other_provider_identifier_state_27,
            npi_1.other_provider_identifier_issuer_27,
            npi_1.other_provider_identifier_28,
            npi_1.other_provider_identifier_type_code_28,
            npi_1.other_provider_identifier_state_28,
            npi_1.other_provider_identifier_issuer_28,
            npi_1.other_provider_identifier_29,
            npi_1.other_provider_identifier_type_code_29,
            npi_1.other_provider_identifier_state_29,
            npi_1.other_provider_identifier_issuer_29,
            npi_1.other_provider_identifier_30,
            npi_1.other_provider_identifier_type_code_30,
            npi_1.other_provider_identifier_state_30,
            npi_1.other_provider_identifier_issuer_30,
            npi_1.other_provider_identifier_31,
            npi_1.other_provider_identifier_type_code_31,
            npi_1.other_provider_identifier_state_31,
            npi_1.other_provider_identifier_issuer_31,
            npi_1.other_provider_identifier_32,
            npi_1.other_provider_identifier_type_code_32,
            npi_1.other_provider_identifier_state_32,
            npi_1.other_provider_identifier_issuer_32,
            npi_1.other_provider_identifier_33,
            npi_1.other_provider_identifier_type_code_33,
            npi_1.other_provider_identifier_state_33,
            npi_1.other_provider_identifier_issuer_33,
            npi_1.other_provider_identifier_34,
            npi_1.other_provider_identifier_type_code_34,
            npi_1.other_provider_identifier_state_34,
            npi_1.other_provider_identifier_issuer_34,
            npi_1.other_provider_identifier_35,
            npi_1.other_provider_identifier_type_code_35,
            npi_1.other_provider_identifier_state_35,
            npi_1.other_provider_identifier_issuer_35,
            npi_1.other_provider_identifier_36,
            npi_1.other_provider_identifier_type_code_36,
            npi_1.other_provider_identifier_state_36,
            npi_1.other_provider_identifier_issuer_36,
            npi_1.other_provider_identifier_37,
            npi_1.other_provider_identifier_type_code_37,
            npi_1.other_provider_identifier_state_37,
            npi_1.other_provider_identifier_issuer_37,
            npi_1.other_provider_identifier_38,
            npi_1.other_provider_identifier_type_code_38,
            npi_1.other_provider_identifier_state_38,
            npi_1.other_provider_identifier_issuer_38,
            npi_1.other_provider_identifier_39,
            npi_1.other_provider_identifier_type_code_39,
            npi_1.other_provider_identifier_state_39,
            npi_1.other_provider_identifier_issuer_39,
            npi_1.other_provider_identifier_40,
            npi_1.other_provider_identifier_type_code_40,
            npi_1.other_provider_identifier_state_40,
            npi_1.other_provider_identifier_issuer_40,
            npi_1.other_provider_identifier_41,
            npi_1.other_provider_identifier_type_code_41,
            npi_1.other_provider_identifier_state_41,
            npi_1.other_provider_identifier_issuer_41,
            npi_1.other_provider_identifier_42,
            npi_1.other_provider_identifier_type_code_42,
            npi_1.other_provider_identifier_state_42,
            npi_1.other_provider_identifier_issuer_42,
            npi_1.other_provider_identifier_43,
            npi_1.other_provider_identifier_type_code_43,
            npi_1.other_provider_identifier_state_43,
            npi_1.other_provider_identifier_issuer_43,
            npi_1.other_provider_identifier_44,
            npi_1.other_provider_identifier_type_code_44,
            npi_1.other_provider_identifier_state_44,
            npi_1.other_provider_identifier_issuer_44,
            npi_1.other_provider_identifier_45,
            npi_1.other_provider_identifier_type_code_45,
            npi_1.other_provider_identifier_state_45,
            npi_1.other_provider_identifier_issuer_45,
            npi_1.other_provider_identifier_46,
            npi_1.other_provider_identifier_type_code_46,
            npi_1.other_provider_identifier_state_46,
            npi_1.other_provider_identifier_issuer_46,
            npi_1.other_provider_identifier_47,
            npi_1.other_provider_identifier_type_code_47,
            npi_1.other_provider_identifier_state_47,
            npi_1.other_provider_identifier_issuer_47,
            npi_1.other_provider_identifier_48,
            npi_1.other_provider_identifier_type_code_48,
            npi_1.other_provider_identifier_state_48,
            npi_1.other_provider_identifier_issuer_48,
            npi_1.other_provider_identifier_49,
            npi_1.other_provider_identifier_type_code_49,
            npi_1.other_provider_identifier_state_49,
            npi_1.other_provider_identifier_issuer_49,
            npi_1.other_provider_identifier_50,
            npi_1.other_provider_identifier_type_code_50,
            npi_1.other_provider_identifier_state_50,
            npi_1.other_provider_identifier_issuer_50,
            npi_1.is_sole_proprietor,
            npi_1.is_organization_subpart,
            npi_1.parent_organization_lbn,
            npi_1.parent_organization_tin,
            npi_1.authorized_official_name_prefix_text,
            npi_1.authorized_official_name_suffix_text,
            npi_1.authorized_official_credential_text,
            npi_1.healthcare_provider_taxonomy_group_1,
            npi_1.healthcare_provider_taxonomy_group_2,
            npi_1.healthcare_provider_taxonomy_group_3,
            npi_1.healthcare_provider_taxonomy_group_4,
            npi_1.healthcare_provider_taxonomy_group_5,
            npi_1.healthcare_provider_taxonomy_group_6,
            npi_1.healthcare_provider_taxonomy_group_7,
            npi_1.healthcare_provider_taxonomy_group_8,
            npi_1.healthcare_provider_taxonomy_group_9,
            npi_1.healthcare_provider_taxonomy_group_10,
            npi_1.healthcare_provider_taxonomy_group_11,
            npi_1.healthcare_provider_taxonomy_group_12,
            npi_1.healthcare_provider_taxonomy_group_13,
            npi_1.healthcare_provider_taxonomy_group_14,
            npi_1.healthcare_provider_taxonomy_group_15,
            npi_1.better_doctor_json_text_old,
            npi_1.bing_images,
            npi_1.google_images,
            npi_1.better_doctor_json_backup,
            npi_1.better_doctor_jsonb
           FROM npi npi_1
          WHERE (npi_1.entity_type_code = '1'::text)) npi
     JOIN derms ON ((derms.taxonomy_code_npi = npi.npi)))
  WITH NO DATA;


--
-- Name: referrals_2009; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE referrals_2009 (
    referrer_npi text,
    referree_npi text,
    pair_count integer,
    unique_beneficiaries_count integer,
    same_day_referrals_count integer
);


--
-- Name: referrals_2015; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE referrals_2015 (
    referrer_npi text,
    referree_npi text,
    pair_count integer,
    unique_beneficiaries_count integer,
    same_day_referrals_count integer
);


--
-- Name: referrals_snapshot_2015; Type: MATERIALIZED VIEW; Schema: public; Owner: -; Tablespace: 
--

CREATE MATERIALIZED VIEW referrals_snapshot_2015 AS
 SELECT referrals_2015.referree_npi,
    sum(referrals_2015.pair_count) AS aggregate_count,
    count(DISTINCT referrals_2015.referrer_npi) AS unique_referrers
   FROM ((referrals_2015
     JOIN npi ON ((npi.npi = referrals_2015.referree_npi)))
     JOIN taxonomy_code ON (((((taxonomy_code.npi = npi.npi) AND (taxonomy_code.code = '207N00000X'::text)) AND (taxonomy_code.is_primary = true)) AND (npi.entity_type_code = '1'::text))))
  GROUP BY referrals_2015.referree_npi
  ORDER BY sum(referrals_2015.pair_count)
  WITH NO DATA;


--
-- Name: yelp_data; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE yelp_data (
    npi text NOT NULL,
    ordinal integer NOT NULL,
    business_id text NOT NULL,
    url text NOT NULL,
    reviews integer NOT NULL,
    rating double precision NOT NULL
);


--
-- Name: cities_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY cities
    ADD CONSTRAINT cities_pkey PRIMARY KEY (geonameid);


--
-- Name: doctor_qa_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY doctor_qa
    ADD CONSTRAINT doctor_qa_pkey PRIMARY KEY (npi);


--
-- Name: better_doctor_jsonb_images; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX better_doctor_jsonb_images ON npi USING btree (((better_doctor_jsonb #>> '{data,profile,image_url}'::text[])));


--
-- Name: business_geocode_index; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX business_geocode_index ON business_geocode USING gist (geom);


--
-- Name: business_geocode_index_geo; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX business_geocode_index_geo ON business_geocode USING gist (geog);


--
-- Name: cities_index; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX cities_index ON cities USING gist (geom);


--
-- Name: cities_index_geo; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX cities_index_geo ON cities USING gist (geog);


--
-- Name: city_state; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX city_state ON npi USING btree (provider_business_practice_location_address_city_name, provider_business_practice_location_address_state_name);


--
-- Name: derms_only_key; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX derms_only_key ON npi_derms_only USING btree (npi);


--
-- Name: npi_better_doctor_jsonb_idx; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX npi_better_doctor_jsonb_idx ON npi USING gin (better_doctor_jsonb);


--
-- Name: npi_key; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX npi_key ON npi USING btree (npi);


--
-- Name: npi_state_taxonomy_query; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX npi_state_taxonomy_query ON npi USING btree (provider_business_practice_location_address_country_code, provider_business_practice_location_address_state_name, healthcare_provider_taxonomy_code_1);


--
-- Name: npi_unique_index; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE UNIQUE INDEX npi_unique_index ON npi USING btree (npi);


--
-- Name: taxonomy_key; Type: INDEX; Schema: public; Owner: -; Tablespace: 
--

CREATE INDEX taxonomy_key ON taxonomy_code USING btree (npi, code, is_primary);


--
-- Name: business_geocode_npi_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY business_geocode
    ADD CONSTRAINT business_geocode_npi_fkey FOREIGN KEY (npi) REFERENCES npi(npi);


--
-- Name: doctor_qa_npi_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY doctor_qa
    ADD CONSTRAINT doctor_qa_npi_fkey FOREIGN KEY (npi) REFERENCES npi(npi);


--
-- Name: taxonomy_code_npi_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY taxonomy_code
    ADD CONSTRAINT taxonomy_code_npi_fkey FOREIGN KEY (npi) REFERENCES npi(npi);


--
-- Name: yelp_data_npi_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY yelp_data
    ADD CONSTRAINT yelp_data_npi_fkey FOREIGN KEY (npi) REFERENCES npi(npi);


--
-- Name: public; Type: ACL; Schema: -; Owner: -
--

REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM spruce;
GRANT ALL ON SCHEMA public TO spruce;
GRANT ALL ON SCHEMA public TO PUBLIC;


--
-- PostgreSQL database dump complete
--

