/** @jsx React.DOM */

function staticURL(path) {
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

	getParameterByName: function(name) {
		name = name.replace(/[\[]/, "\\[").replace(/[\]]/, "\\]");
		var regex = new RegExp("[\\?&]" + name + "=([^&#]*)"),
			results = regex.exec(location.search);
		return results == null ? "" : decodeURIComponent(results[1].replace(/\+/g, " "));
	},

	ancestorWithClass: function(el, className) {
		while (el != document && !el.classList.contains(className)) {
			el = el.parentNode;
		}
		if (el == document) {
			el = null;
		}
		return el;
	},

	swallowEvent: function(e) {
		e.preventDefault();
		return false;
	},

	formatEmailAddress: function(name, email) {
		// TODO: don't always need the quotes around name
		return '"' + name + '" <' + email + '>';
	},

	unixTimestampToDate: function(unixTS) {
		return new Date(unixTS*1000);
	},

	// Find the right method, call on correct element
	fullscreen: function(element) {
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

	staticURL: staticURL,

	Alert: React.createClass({displayName: "Alert",
		propTypes: {
			type: React.PropTypes.oneOf(['success', 'info', 'warning', 'danger'])
		},
		getDefaultProps: function() {
			return {"type": "info"};
		},
		render: function() {
			if (this.props.children.length == 0) {
				return null;
			}
			return <div className={"alert alert-"+this.props.type} role="alert">{this.props.children}</div>;
		}
	}),

	LoadingAnimation: React.createClass({displayName: "LoadingAnimation",
		render: function() {
			return <img src={staticURL("/img/loading.gif")} />;
		}
	})
}

// Polyfill from https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Array/filter
if (!Array.prototype.filter) {
	Array.prototype.filter = function(fun /*, thisArg */) {
		"use strict";

		if (this === void 0 || this === null) {
			throw new TypeError();
		}

		var t = Object(this);
		var len = t.length >>> 0;
		if (typeof fun !== "function") {
			throw new TypeError();
		}

		var res = [];
		var thisArg = arguments.length >= 2 ? arguments[1] : void 0;
		for (var i = 0; i < len; i++) {
			if (i in t) {
				var val = t[i];

				// NOTE: Technically this should Object.defineProperty at
				//       the next index, as push can be affected by
				//       properties on Object.prototype and Array.prototype.
				//       But that method's new, and collisions should be
				//       rare, so use the more-compatible alternative.
				if (fun.call(thisArg, val, i, t)) {
					res.push(val);
				}
			}
		}

		return res;
	};
}
