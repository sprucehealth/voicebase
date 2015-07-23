/* @flow */

var React = require("react");

function staticURL(path: string): string {
	return Spruce.BaseStaticURL + path
}

module.exports = {
	states: [
		{name: "Select State", value: ""},
		{name: "Alabama", value: "AL"},
		{name: "Alaska", value: "AK"},
		{name: "Arizona", value: "AZ"},
		{name: "Arkansas", value: "AR"},
		{name: "California", value: "CA"},
		{name: "Colorado", value: "CO"},
		{name: "Connecticut", value: "CT"},
		{name: "Delaware", value: "DE"},
		{name: "Florida", value: "FL"},
		{name: "Georgia", value: "GA"},
		{name: "Hawaii", value: "HI"},
		{name: "Idaho", value: "ID"},
		{name: "Illinois", value: "IL"},
		{name: "Indiana", value: "IN"},
		{name: "Iowa", value: "IA"},
		{name: "Kansas", value: "KS"},
		{name: "Kentucky", value: "KY"},
		{name: "Louisiana", value: "LA"},
		{name: "Maine", value: "ME"},
		{name: "Maryland", value: "MD"},
		{name: "Massachusetts", value: "MA"},
		{name: "Michigan", value: "MI"},
		{name: "Minnesota", value: "MN"},
		{name: "Mississippi", value: "MS"},
		{name: "Missouri", value: "MO"},
		{name: "Montana", value: "MT"},
		{name: "Nebraska", value: "NE"},
		{name: "Nevada", value: "NV"},
		{name: "New Hampshire", value: "NH"},
		{name: "New Jersey", value: "NJ"},
		{name: "New Mexico", value: "NM"},
		{name: "New York", value: "NY"},
		{name: "North Carolina", value: "NC"},
		{name: "North Dakota", value: "ND"},
		{name: "Ohio", value: "OH"},
		{name: "Oklahoma", value: "OK"},
		{name: "Oregon", value: "OR"},
		{name: "Pennsylvania", value: "PA"},
		{name: "Rhode Island", value: "RI"},
		{name: "South Carolina", value: "SC"},
		{name: "South Dakota", value: "SD"},
		{name: "Tennessee", value: "TN"},
		{name: "Texas", value: "TX"},
		{name: "Utah", value: "UT"},
		{name: "Vermont", value: "VT"},
		{name: "Virginia", value: "VA"},
		{name: "Washington", value: "WA"},
		{name: "Washington, D.C.", value: "DC"},
		{name: "West Virginia", value: "WV"},
		{name: "Wisconsin", value: "WI"},
		{name: "Wyoming", value: "WY"}
	],

	deleteProperties: function(obj: any, props: Array<string>): void {
		for (var i = 0; i < props.length; i++) {
			delete(obj[props[i]])
		}
	},

	getParameterByName: function(name: string): string {
		name = name.replace(/[\[]/, "\\[").replace(/[\]]/, "\\]");
		var regex = new RegExp("[\\?&]" + name + "=([^&#]*)"),
			results = regex.exec(location.search);
		return results == null ? "" : decodeURIComponent(results[1].replace(/\+/g, " "));
	},

	ancestorWithClass: function(el: any, className: string): any {
		while (el != document && !el.classList.contains(className)) {
			el = el.parentNode;
		}
		if (el == document) {
			el = null;
		}
		return el;
	},

	swallowEvent: function(e: any): void {
		e.preventDefault();
	},

	formatEmailAddress: function(name: string, email: string): string {
		// TODO: don't always need the quotes around name
		return '"' + name + '" <' + email + '>';
	},

	unixTimestampToDate: function(unixTS: number): Date {
		return new Date(unixTS*1000);
	},

	// Find the right method, call on correct element
	fullscreen: function(element: any): void {
		if(element.requestFullscreen) {
			element.requestFullscreen();
		} else if(element.mozRequestFullScreen) {
			element.mozRequestFullScreen();
		} else if(element.webkitRequestFullscreen) {
			element.webkitRequestFullscreen();
		} else if(element.msRequestFullscreen) {
			element.msRequestFullscreen();
		}
	},

	timeSince: function(startEpoch: number, endEpoch: number): string {
		var yearsAgo = 0;
		var weeksAgo = 0;
		var daysAgo = 0;
		var hoursAgo = 0;
		var minutesAgo = 0;
		var minute = 60;
		var hour = 60 * minute;
		var day = 24 * hour;
		var week = 7 * day;
		var year = 365 * day;
		var deltaSecs = endEpoch - startEpoch;
		var descriptors = []
		if(deltaSecs > year) {
			yearsAgo = Math.floor(deltaSecs/year)
			deltaSecs -= year * yearsAgo
		}
		if(deltaSecs > week) {
			weeksAgo = Math.floor(deltaSecs/week)
			deltaSecs -= week * weeksAgo
		}
		if(deltaSecs > day) {
			daysAgo = Math.floor(deltaSecs/day)
			deltaSecs -= day * daysAgo
		}
		if(deltaSecs > hour) {
			hoursAgo = Math.floor(deltaSecs/hour)
			deltaSecs -= hour * hoursAgo
		}
		if(deltaSecs > minute) {
			minutesAgo = Math.floor(deltaSecs/minute)
			deltaSecs -= minute * minutesAgo
		}
		if(yearsAgo != 0){
			descriptors.push(yearsAgo + " " + (yearsAgo == 1 ? "year" : "years"))
		}
		if(weeksAgo != 0){
			descriptors.push(weeksAgo + " " + (weeksAgo == 1 ? "week" : "weeks"))
		}
		if(daysAgo != 0){
			descriptors.push(daysAgo + " " + (daysAgo == 1 ? "day" : "days"))
		}
		if(hoursAgo != 0){
			descriptors.push(hoursAgo + " " + (hoursAgo == 1 ? "hour" : "hours"))
		}
		if(minutesAgo != 0){
			descriptors.push(minutesAgo + " " + (minutesAgo == 1 ? "minute" : "minutes"))
		}
		return descriptors.join(" ") + " ago"
	},

	isInteger: function(str: string): boolean {
		var n = ~~Number(str);
		return String(n) === str && n >= 0;
	},

	getQueryParams: function(): any {
		var params = {};
		var query = window.location.search.substring(1);
		if(query.length > 1) {
			var vars = query.split('&');
			for (var i = 0; i < vars.length; i++) {
					var pair = vars[i].split('=');
					var key = decodeURIComponent(pair[0]) 
					if(!params[key]){
						params[key] = []
					}
					params[key].push(decodeURIComponent(pair[1]))
			}
		}
		return params;
	},

	staticURL: staticURL,

	Alert: React.createClass({displayName: "Alert",
		propTypes: {
			type: React.PropTypes.oneOf(['success', 'info', 'warning', 'danger']).isRequired
		},
		getDefaultProps: function(): any {
			return {"type": "info"};
		},
		render: function(): any {
			if (this.props.children.length == 0) {
				return null;
			}
			return <div className={"alert alert-"+this.props.type} role="alert">{this.props.children}</div>;
		}
	}),

	LoadingAnimation: React.createClass({displayName: "LoadingAnimation",
		render: function(): any {
			return <img src={staticURL("/img/loading.gif")} />;
		}
	})
}
