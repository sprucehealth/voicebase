/** @jsx React.DOM */

function staticURL(path) {
	return Spruce.BaseStaticURL + path
}

module.exports = {
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
