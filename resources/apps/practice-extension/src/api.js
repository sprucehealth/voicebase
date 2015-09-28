/* @flow */

let JSONHeaders = {
	'Accept': 'application/json',
	'Content-Type': 'application/json'
}

export function submitDemoRequest(firstName: string, lastName: string, email: string, phone: string, state: string): Promise {
	return fetch('/api/practices/demo-request', {
		method: 'post',
		headers: JSONHeaders,
		body: JSON.stringify({
			first_name: firstName,
			last_name: lastName,
			email: email,
			phone: phone,
			state: state,
		})
	})
}

export function submitWhitepaperRequest(firstName: string, lastName: string, email: string): Promise {
	return fetch('/api/practices/whitepaper-request', {
		method: 'post',
		headers: JSONHeaders,
		body: JSON.stringify({
			first_name: firstName,
			last_name: lastName,
			email: email,
		})
	})
}