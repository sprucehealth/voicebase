/* @flow */

export function signup(firstName: string, lastName: string, email: string, licensedLocations: string, reasonsInterested: string, dermatologyInterests: string, referralSource: string): Promise {
	return fetch('/submit', {
		method: 'post',
		headers: {
			'Accept': 'application/json',
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({
			first_name: firstName,
			last_name: lastName,
			email: email,
			licensed_locations: licensedLocations,
			reasons_interested: reasonsInterested,
			dermatology_interests: dermatologyInterests,
			referral_source: referralSource,
		})
	})
}