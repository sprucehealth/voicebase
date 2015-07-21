/* @flow */

var React = require("react");
var Utils = require("../../libs/utils.js");
var Constants = require("./Constants.js");

var SignInView = React.createClass({displayName: "SignInView",
	propTypes: {
		onFormSubmit: React.PropTypes.func.isRequired,
	},
	handleSubmit: function(e: any) {
		e.preventDefault();
		this.props.onFormSubmit({})
	},
	render: function(): any {
		return (
			<div>
			</div>
		)
	}
});

module.exports = SignInView;