/* @flow */

module.exports = {
	/*
	Available properties:
		error: string
		screen_id: string
		time_spent: number
		app_type: string
	*/
	record: function(eventName: string, properties: ?any, sync: ?bool) {
		sync = sync || false
		var req = new XMLHttpRequest();
		req.onerror = function(err: any) {
			console.error(err);
		}
		req.open('POST', '/api/events', !sync);
		req.setRequestHeader("Content-Type", "application/json");
		req.send(JSON.stringify({
			current_time: Date.now() / 1000.0,
			events: [
				{event: eventName, properties: properties},
			],
		}))
	},
};
