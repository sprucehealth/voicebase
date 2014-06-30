"""
--------------------------------------------------------------------------------

Requirements:

- Internet connection
- Python 2.7 (https://www.python.org/downloads/)
- Active LiveAddress API subscription (http://smartystreets.com/pricing)
- Valid Secret Key Pair (http://smartystreets.com/account/keys)

--------------------------------------------------------------------------------

This program preserves the core functionality of the deprecated LiveAddress
for Lists service by translating the output of the LiveAddress API into a
text file (the 'everything' file) but without concerning itself with much of
the backend infrastructure associated with the deprecated service (asynchronous
messaging infrastructure, a fleet of auto-scaled processing servers, queueing of
lists, web GUI, free file storage). Our reason for writing this code was to
illustrate that our LiveAddress for Lists service was, in essence, a
well-crafted example of how to use the LiveAddress API to do something
interesting. In the case of LiveAddress for Lists, the "interesting" thing
was to process a file with address data.

IMPORTANT NOTE: This program is provided at no charge to you as an example.
It is not to be distributed beyond your organization. You are welcome to use
it, to modify it, to extend it, and even to translate it to other languages.
We assume no responsibility for maintenance, fixes, or modifications to this
code or derivative works.

The input file must be a TAB-DELIMITED text file with a header row that has
column names from the following set (case-insensitive):

[
    'Id', 'FullName', 'FirstName', 'LastName', 'OrganizationName',
    'Street1', 'Secondary', 'Street2', 'City', 'State', 'ZipCode',
    'CityStateZipCode', 'Plus4Code', 'Urbanization'
]

The column names may be presented in any order, but each record's field data
must match that order. Additional fields are welcome as well, but they won't be
included in the requests to the LiveAddress API.

The only combination of fields that is required is one of the following:

Street1, ZipCode
Street1, City, State
Street1, City, State, ZipCode
Street1, LastLine (city/state/zipcode in a single field)

The output of this program is a file that resembles the "everything" file
produced by the deprecated LiveAddress for Lists service.

    Usage: python list_processing.py <path-to-input-file> <path-to-output-file>

The only known differences between the output of this program and the
LiveAddress for Lists service are represented by the absence of the following
columns:

    - Freeform
    - FlagReason
    - LACSInd

--------------------------------------------------------------------------------

It should be noted that this program contains logic that we've used at
SmartyStreets for some time now. Much of the logic (like de-duplication and
deciding which fields to use for the 'addressee') are general purpose. You might
have good reasons to rewrite portions of this program in a way that matches your
use case more specifically.

If you are interested in modifying or extending this program to produce the
other files that were included in the LiveAddress for Lists service, here are
basic instructions:

mailable file: take all instances of AddressResponse where:
    - self.deliverable == 'Y' and self.duplicate != 'Y'

rejected file: take all instances of AddressResponse where:
    - self.deliverable != 'Y' and self.duplicate != 'Y'

duplicates file: take all instances of AddressResponse where:
    - self.duplicate == 'Y'

components file: take all instances of AddressResponse and dump the
                 following fields out:
    - Sequence
    - Duplicate
    - Deliverable
    - PrimaryNumber
    - StreetName
    - StreetPredirection
    - StreetPostdirection
    - StreetSuffix
    - SecondaryNumber
    - SecondaryDesignator
    - ExtraSecondaryNumber
    - ExtraSecondaryDesignator
    - PmbDesignator
    - PmbNumber
    - CityName
    - DefaultCityName
    - StateAbbreviation
    - ZipCode
    - Plus4Code
    - DeliveryPoint
    - DeliveryPointCheckDigit
    - Urbanization

    Some of these fields may have to be extracted from the JSON response. See
    the documentation below for details on the 'components' structure.

--------------------------------------------------------------------------------

LiveAddress API Documentation:

- http://smartystreets.com/kb/liveaddress-api/rest-endpoint
- http://smartystreets.com/kb/liveaddress-api/parsing-the-response
- http://smartystreets.com/kb/liveaddress-api/field-definitions

--------------------------------------------------------------------------------

Enjoy!
"""


I_UNDERSTAND_AND_AGREE_TO_THE_TERMS_IN_THE_MODULE_DOC_STRING_ABOVE = True


import argparse
import hashlib
import httplib
import io
import json
import os
import traceback
import urllib
import urllib2

# NOTE: config.py is where the authentication credentials are currently stored:
import config


def main(input_file, output_file):
    header_line, headers = analyze(input_file)
    if headers is None:
        exit(1)

    with io.open(input_file) as records, io.open(output_file, 'w') as output:
        write_header_row(header_line, output)
        process(records, headers, output)
        output.flush()


################################################################################
# Analysis of file header and contents                                         #
################################################################################


def analyze(input_file):
    """
    Does the input file exist?
    Does it have a header row with the right kind of field names?
    Does each record have the appropriate number of fields?
    """

    if not os.path.exists(input_file):
        print 'The given input file does not exist!'
        return None, None

    header_line = None
    headers = None
    with io.open(input_file) as records:
        columns = 0
        header = ''

        for number, line in enumerate(records, start=1):
            if number == 1:
                header = line
                fields = line.split(TAB)
                columns = len(fields)
                headers = identify_headers(fields)
                if headers is None:
                    return None, None
                else:
                    header_line = line

            field_count = line.count(TAB) + 1
            if field_count != columns:
                print 'Line number {0} had {1} fields but each record ' \
                      'must have {2} fields (separated by tabs) to ' \
                      'match the header row: "{3}"' \
                    .format(number, field_count, columns, header)
                return None, None

    return header_line, headers


def identify_headers(fields):
    """
    Required: Street1 & (ZIP_CODE | CITY & STATE | LAST_LINE)
    """

    headers = dict.fromkeys(Headers.field_keys())
    for index, field in enumerate(fields):
        field = field.strip().lower()
        headers[field] = index

    for header in headers.keys():
        if headers[header] is None:
            del headers[header]

    if headers.get(Headers.STREET1) is None:
        print 'You must include a "Street1" field in the header row!'
        return None

    if headers.get(Headers.ZIP_CODE) is not None:
        return headers

    if headers.get(Headers.LAST_LINE) is not None:
        return headers

    if (headers.get(Headers.CITY) is not None and
            headers.get(Headers.STATE) is not None):
        return headers

    print 'At minimum, you must include "Street1" and at least one ' \
          'of the following combinations:\n\t' \
          '"ZipCode"\n\t' \
          '"CityStateZipCode"\n\t' \
          '"City", "State"'
    return None


def write_header_row(header_line, output):
    """
    Inserts the headers from the input file into the output file header row.
    """

    header_fields = header_line.strip().split(TAB)
    header_fields = TAB.join(['[{0}]'.format(x) for x in header_fields])
    output_header_row = Templates.header.format(
        input_headers=header_fields)
    output.write(output_header_row)


################################################################################
# Processing of address records                                                #
################################################################################


def process(records, headers, output):
    """
    Each HTTP request payload is a batch of <= 100 addresses (json).
    HTTP error (other than 400) causes unhandled exception (program exits).
    """

    batch = []
    hashes = set()

    for number, line in enumerate(records):
        if not number:
            continue

        input_record = InputRecord(line, headers)
        if len(batch) >= 100:
            verified_batch, ok = verify(batch)
            if not ok:
                return

            write_batch(verified_batch, hashes, output)
            batch[:] = []

        batch.append(input_record)

    if len(batch):
        verified_batch, ok = verify(batch)
        if not ok:
            return

        write_batch(verified_batch, hashes, output)


def is_duplicate(hashes, record):
    """
    Each address result is concatenated and hashed using MD5. `hashes` set
    maintains record of everything we've already seen.
    """

    value = record.hash_input()
    if value == BLANK_ADDRESS_CONCATENATION:
        return False

    hasher = hashlib.md5()
    hasher.update(value)
    hashed = hasher.digest()

    if hashed in hashes:
        return True

    hashes.add(hashed)
    return False


def verify(batch):
    """
    LiveAddress API - HTTP nitty-gritty.
    """

    records = sanitize(batch)
    payload = json.dumps(records)
    url = HTTP.api_url + '?' + urllib.urlencode(config.AUTHENTICATION)
    request = urllib2.Request(url, payload, HTTP.request_headers)
    first = batch[0].sequence
    last = batch[-1].sequence

    # Disables PyCharm warning:
    # noinspection PyBroadException
    try:
        response = urllib2.urlopen(request)
        return parse(response.read(), batch), True
    except urllib2.HTTPError, e:
        if e.code == 400:
            print 'Bad batch from records {0}-{1}, returning ' \
                  'empty result set... ({2})'.format(first, last, e.reason)
            print payload
            return [], True
        else:
            print 'urllib2.HTTPError: HTTP Status Code for batch of records ' \
                  '({0}-{1}): {2} ({3})'.format(first, last, e.code, e.reason)
            print payload
            return [], False
    except urllib2.URLError, e:
        print 'urllib2.URLError for batch of records ' \
              '({0}-{1}):\n\tMessage: {2}\n\tReason: {3}' \
            .format(first, last, e.message, e.reason)
        print payload
        return [], False
    except httplib.HTTPException, e:
        print 'httplib.HTTPException for batch of records ' \
              '({0}-{1}): {2}'.format(first, last, e.message)
        print payload
        return [], False
    except Exception:
        print 'Exception for batch of records ({0}-{1}): {2}' \
            .format(first, last, traceback.format_exc())
        print payload
        return [], False


def sanitize(input_batch):
    """
    Make sure the JSON payload will have the correct fields (and not
    any extra fields).
    """

    records = [x.to_dict() for x in input_batch]
    for record in records:
        if not record.get('street'):
            record['street'] = '<unknown_street>'
            # response: '<Unknown_Street>'

        if (not record.get('zipcode')
                and not record.get('state')
                and not record.get('city')
                and not record.get('lastline')):
            record['lastline'] = '<lastline_unknown>'
            # response: 'Lastline Unknown'

    for record in records:
        for field in record.keys():
            if not record[field].strip():
                del record[field]

    return records


def parse(json_result, input_batch):
    """
    Put the resulting JSON response into a well-formed structure.
    """

    results = json.loads(json_result)
    batch = []
    for item in input_batch:
        processed = AddressResponse()
        processed.include_input(item)
        batch.append(processed)

    for index, result in enumerate(results):
        batch_index = int(result['input_index'])
        batch[batch_index].include_output(result)

    return batch


def write_batch(batch, hashes, output):
    for record in batch:
        if is_duplicate(hashes, record):
            record.duplicate = 'Y'

        output.write(unicode(record))


################################################################################
# Helpers and constants                                                        #
################################################################################


QUOTE = '"'
TAB = u'\t'
BLANK_ADDRESS_CONCATENATION = '/////'


class InputRecord(object):
    """
    The InputRecord defines the fields that will be submitted in JSON format
    to the LiveAddress API.
    """

    sequence_counter = 1

    def __unicode__(self):
        return TAB.join(x.strip().replace(QUOTE, '')
                        for x in self.original.split(TAB))

    def to_dict(self):
        return {
            # These key names must match the expected
            # LiveAddress API JSON input fields:
            'input_id': self.input_id,
            'street': self.street,
            'street2': self.street2,
            'city': self.city,
            'state': self.state,
            'zipcode': self.zipcode,
            'plus4': self.plus4,
            'urbanization': self.urbanization,
            'secondary': self.secondary,
            'lastline': self.lastline,
            'addressee': self.addressee,
        }

    def __init__(self, line, headers):
        self.original = line
        self.sequence = InputRecord.sequence_counter
        self.fields = [x.strip() for x in line.split(TAB)]

        self.input_id = get(self.fields, headers.get(Headers.ID))
        self.street = get(self.fields, headers.get(Headers.STREET1))
        self.street2 = get(self.fields, headers.get(Headers.STREET2))
        self.city = get(self.fields, headers.get(Headers.CITY))
        self.state = get(self.fields, headers.get(Headers.STATE))
        self.zipcode = get(self.fields, headers.get(Headers.ZIP_CODE))
        self.plus4 = get(self.fields, headers.get(Headers.PLUS4CODE))
        self.urbanization = get(self.fields, headers.get(Headers.URBANIZATION))
        self.secondary = get(self.fields, headers.get(Headers.SECONDARY))
        self.lastline = get(self.fields, headers.get(Headers.LAST_LINE))
        self.addressee = self.decide_on_addressee(headers)

        InputRecord.sequence_counter += 1
        report_progress(InputRecord.sequence_counter)

    def decide_on_addressee(self, headers):
        """
        This is the logic we've used at SmartyStreet for some time. But it
        doesn't have to be like this is it's not helpful.
        """
        organization = get(self.fields, headers.get(Headers.ORGANIZATION))
        if organization:
            return organization

        full = get(self.fields, headers.get(Headers.FULL_NAME))
        if full:
            return full

        first = get(self.fields, headers.get(Headers.FIRST_NAME))
        last = get(self.fields, headers.get(Headers.LAST_NAME))
        return (first + ' ' + last).strip()


def get(items, index):
    """
    Safe indexing of list (avoids IndexError and subsequent try-except block).
    Also strips leading and trailing whitespace and leading and trailing quotes.
    """

    if index is None:
        return ''

    if index >= len(items) or index < 0:
        print 'Tried to access an invalid index in the line! ' \
              'Index: {0} | FieldCount: {1}'.format(index, len(items))
        return ''

    return items[index].strip().replace(QUOTE, '')


def report_progress(sequence):
    if not sequence % 1000:
        print sequence


class AddressResponse(object):
    """
    The AddressResponse serves to convert the JSON response from the
    LiveAddress API to the tab-delimited format of the output file.
    Each instance of this class corresponds directly to a record in
    the output file.
    """

    def hash_input(self):
        """
        The value returned is hashed and used to determine duplicate
        addresses across the entire list/file. These are the values we've
        used for some time to de-duplicate. Feel free to modify this method
        if you have a more appropriate implementation for your use case.
        """

        return '{0}/{1}/{2}/{3}/{4}/{5}'.format(
            self.delivery_line_1,
            self.delivery_line_2,
            self.city,
            self.state,
            self.full_zip_code,
            self.urbanization)

    def include_input(self, input_record):
        """
        Keeps track of all input fields as they are included in the output file.
        """

        self.sequence = input_record.sequence
        self.input = input_record

    def include_output(self, structure):
        """
        Logic to convert JSON structure to a tab-delimited record.
        """

        analysis = structure.get('analysis') or {}
        components = structure.get('components') or {}
        metadata = structure.get('metadata') or {}

        self.active = analysis.get('active') or ''
        self.carrier_route = metadata.get('carrier_route') or ''
        self.check_digit = components.get('delivery_point_check_digit') or ''

        self.city = components.get('city_name') or ''
        if self.city == 'Lastline Unknown':
            self.city = ''

        self.cmra = analysis.get('dpv_cmra') or ''
        self.congressional_district = \
            metadata.get('congressional_district') or ''
        self.county_fips = metadata.get('county_fips') or ''
        self.county_name = metadata.get('county_name') or ''
        self.default_flag = metadata.get('building_default_indicator') or ''
        self.deliverable = 'Y' if (
            analysis.get('dpv_match_code') in ['Y', 'S', 'D'] and
            analysis.get('dpv_vacant') == 'N' and
            analysis.get('active') == 'Y') else ''

        self.delivery_line_1 = structure.get('delivery_line_1') or ''
        if self.delivery_line_1.lower() == '<unknown_street>':
            self.delivery_line_1 = ''

        self.delivery_line_2 = structure.get('delivery_line_2') or ''
        self.delivery_point = components.get('delivery_point') or ''
        self.delivery_point_barcode = '/{0}/'.format(
            structure.get('delivery_point_barcode') or '')
        self.dpv_footnotes = analysis.get('dpv_footnotes') or ''
        self.dpv_code = analysis.get('dpv_match_code') or ''
        self.dst = 'Y' if metadata.get('dst') else ''
        self.duplicate = ''  # The value of this field is computed later.
        self.elot_sequence = metadata.get('elot_sequence') or ''
        self.elot_sort = metadata.get('elot_sort') or ''
        self.ews = 'Y' if analysis.get('ews_match') else ''
        self.firmname = structure.get('addressee') or ''
        self.footnotes = analysis.get('footnotes') or ''
        self.full_zip_code = \
            (components.get('zipcode') + '-' + components.get('plus4_code')) \
            if 'zipcode' in components and 'plus4_code' in components \
            else (components.get('zipcode') or '')
        self.lacs_link_code = analysis.get('lacslink_code') or ''
        self.lacs_link_indicator = analysis.get('lacslink_indicator') or ''
        self.latitude = metadata.get('latitude') or '0'
        self.longitude = metadata.get('longitude') or '0'
        self.plus4 = components.get('plus4_code') or ''
        self.pmb_number = components.get('pmb_number') or ''
        self.pmb_unit = components.get('pmb_designator') or ''
        self.precision = metadata.get('precision') or ''
        self.process_flag = 'P' if analysis.get('dpv_match_code') else 'F'
        self.rdi = metadata.get('rdi') or 'None'
        self.record_type = metadata.get('record_type') or ''
        self.state = components.get('state_abbreviation') or ''
        self.suite_link_match = 'Y' if analysis.get('suitelink_match') else ''
        self.time_zone = metadata.get('time_zone') or ''
        self.urbanization = components.get('urbanization') or ''
        self.utc_offset = str(int(metadata.get('utc_offset') or '0'))
        self.vacant = analysis.get('dpv_vacant') or ''
        self.zip_code = components.get('zipcode') or ''
        self.zip_type = metadata.get('zip_type') or ''

    def _to_dict(self):
        return {
            # These key names must match those in the Template.row!
            'active': self.active,
            'carrier_route': self.carrier_route,
            'check_digit': self.check_digit,
            'city': self.city,
            'cmra': self.cmra,
            'congressional_district': self.congressional_district,
            'county_fips': self.county_fips,
            'county_name': self.county_name,
            'default_flag': self.default_flag,
            'deliverable': self.deliverable,
            'delivery_line_1': self.delivery_line_1,
            'delivery_line_2': self.delivery_line_2,
            'delivery_point': self.delivery_point,
            'delivery_point_barcode': self.delivery_point_barcode,
            'dpv_footnotes': self.dpv_footnotes,
            'dpv_code': self.dpv_code,
            'dst': self.dst,
            'duplicate': self.duplicate,
            'elot_sequence': self.elot_sequence,
            'elot_sort': self.elot_sort,
            'ews': self.ews,
            'firmname': self.firmname,
            'footnotes': self.footnotes,
            'full_zip_code': self.full_zip_code,
            'lacs_link_code': self.lacs_link_code,
            'lacs_link_indicator': self.lacs_link_indicator,
            'latitude': self.latitude,
            'longitude': self.longitude,
            'plus4': self.plus4,
            'pmb_number': self.pmb_number,
            'pmb_unit': self.pmb_unit,
            'precision': self.precision,
            'process_flag': self.process_flag,
            'rdi': self.rdi,
            'record_type': self.record_type,
            'sequence': self.sequence,
            'state': self.state,
            'suite_link_match': self.suite_link_match,
            'time_zone': self.time_zone,
            'urbanization': self.urbanization,
            'utc_offset': self.utc_offset,
            'vacant': self.vacant,
            'zip_code': self.zip_code,
            'zip_type': self.zip_type,
        }

    def __unicode__(self):
        """
        Turns self into a tab-delimited unicode string for the output file.
        """

        output = Templates.row.format(**self._to_dict())
        with_input = output.format(all_input_fields=unicode(self.input))
        return with_input

    def __init__(self):
        self.input = None
        self.active = ''
        self.carrier_route = ''
        self.check_digit = ''
        self.city = ''
        self.cmra = ''
        self.congressional_district = ''
        self.county_fips = ''
        self.county_name = ''
        self.default_flag = ''
        self.deliverable = ''
        self.delivery_line_1 = ''
        self.delivery_line_2 = ''
        self.delivery_point = ''
        self.delivery_point_barcode = ''
        self.dpv_footnotes = ''
        self.dpv_code = ''
        self.dst = ''
        self.duplicate = ''
        self.elot_sequence = ''
        self.elot_sort = ''
        self.ews = ''
        self.firmname = ''
        self.footnotes = ''
        self.full_zip_code = ''
        self.lacs_link_code = ''
        self.lacs_link_indicator = ''
        self.latitude = ''
        self.longitude = ''
        self.plus4 = ''
        self.pmb_number = ''
        self.pmb_unit = ''
        self.precision = ''
        self.process_flag = ''
        self.rdi = ''
        self.record_type = ''
        self.sequence = ''
        self.state = ''
        self.suite_link_match = ''
        self.time_zone = ''
        self.urbanization = ''
        self.utc_offset = ''
        self.vacant = ''
        self.zip_code = ''
        self.zip_type = ''


class Headers(object):
    ID = 'Id'.lower()
    FULL_NAME = 'FullName'.lower()
    FIRST_NAME = 'FirstName'.lower()
    LAST_NAME = 'LastName'.lower()
    ORGANIZATION = 'OrganizationName'.lower()
    STREET1 = 'Street1'.lower()
    STREET2 = 'Street2'.lower()
    SECONDARY = 'Secondary'.lower()
    CITY = 'City'.lower()
    STATE = 'State'.lower()
    ZIP_CODE = 'ZipCode'.lower()
    LAST_LINE = 'CityStateZipCode'.lower()
    PLUS4CODE = 'Plus4Code'.lower()
    URBANIZATION = 'Urbanization'.lower()

    @classmethod
    def field_keys(cls):
        return [
            cls.ID,

            cls.FULL_NAME, cls.FIRST_NAME, cls.LAST_NAME, cls.ORGANIZATION,

            cls.STREET1, cls.STREET2, cls.SECONDARY,

            cls.CITY, cls.STATE, cls.ZIP_CODE,
            cls.LAST_LINE, cls.PLUS4CODE,

            cls.URBANIZATION,
        ]


class HTTP(object):
    api_url = 'https://api.smartystreets.com/street-address'
    request_headers = {
        'Content-Type': 'application/json',

        'x-include-invalid': 'true',  # this non-standard header ensures that
        # we get data back for addresses that fail USPS DPV (delivery
        # point validation)
    }


class Templates(object):
    header = TAB.join([
        'Sequence', 'Duplicate', 'Deliverable', '{input_headers}', 'FirmName',
        'DeliveryLine1', 'DeliveryLine2', 'Urbanization', 'City', 'State',
        'FullZIPCode', 'ZIPCode', 'AddonCode', 'PMBUnit', 'PMBNumber',
        'ProcessFlag', 'Footnotes', 'EWS', 'CountyFips', 'CountyName',
        'DPVCode', 'DPVFootnotes', 'CMRA', 'Vacant', 'Active', 'DefaultFlag',
        'LACSLinkCode', 'LACSLinkInd', 'DeliveryPoint', 'CheckDigit',
        'DeliveryPointBarcode', 'CarrierRoute', 'RecordType', 'ZIPType',
        'CongressionalDistrict', 'RDI', 'ELotSequence', 'ELotSort',
        'SuiteLinkMatch', 'TimeZone', 'UTCOffset', 'DST',
        'Latitude', 'Longitude', 'Precision',
    ]) + '\n'

    row = TAB.join([
        '{sequence}', '{duplicate}', '{deliverable}',
        '{{all_input_fields}}',
        '{firmname}', '{delivery_line_1}', '{delivery_line_2}',
        '{urbanization}', '{city}', '{state}', '{full_zip_code}', '{zip_code}',
        '{plus4}', '{pmb_unit}', '{pmb_number}', '{process_flag}',
        '{footnotes}', '{ews}', '{county_fips}', '{county_name}',
        '{dpv_code}', '{dpv_footnotes}', '{cmra}', '{vacant}', '{active}',
        '{default_flag}', '{lacs_link_code}', '{lacs_link_indicator}',
        '{delivery_point}', '{check_digit}', '{delivery_point_barcode}',
        '{carrier_route}', '{record_type}', '{zip_type}',
        '{congressional_district}', '{rdi}', '{elot_sequence}', '{elot_sort}',
        '{suite_link_match}', '{time_zone}', '{utc_offset}', '{dst}',
        '{latitude}', '{longitude}', '{precision}',
    ]) + '\n'


################################################################################
# Startup and argument parsing                                                 #
################################################################################


WARNING = """

--------------------------------------------------------------------------------
--------------------------------------------------------------------------------

ATTENTION:

Before you can use this program, please change the value of the constant
entitled "I_UNDERSTAND_AND_AGREE_TO_THE_TERMS_IN_THE_MODULE_DOC_STRING_ABOVE"
to `True` (or delete the if block that emits this string). But please don't do
that until you have actually read the documentation. (The basic idea is that we
aren't going to maintain or modify this code for you.)

"""


if __name__ == '__main__':
    if not I_UNDERSTAND_AND_AGREE_TO_THE_TERMS_IN_THE_MODULE_DOC_STRING_ABOVE:
        print WARNING
        print __doc__
        print WARNING
        exit(1)

    parser = argparse.ArgumentParser()
    parser.add_argument('input')
    parser.add_argument('output')
    args = parser.parse_args()
    main(args.input, args.output)