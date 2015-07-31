/* @flow */

var React = require("react");

var SubmitButtonView = React.createClass({displayName: "SubmitButtonView",
	propTypes: {
		title: React.PropTypes.string.isRequired,
		appearsDisabled: React.PropTypes.bool.isRequired,
	},
	render: function(): any {
		var buttonStyle = {
			background: "none",
			backgroundColor: (this.props.appearsDisabled ? "rgba(11, 165, 197, 0.3)" : "rgba(11, 165, 197, 1)"), // #0BA5C5
			transition: "background-color 0.75s",
			WebkitTransition: "background-color 0.75s",
			color: "white",
			border: "none",
			height: "52px",
			width: "100%",
			marginTop: "24px",
			marginBottom: "16px",
			fontSize: "16px",
			cursor: "pointer",
			WebkitBorderTopLeftRadius: "26px",
			WebkitBorderTopRightRadius: "26px",
			WebkitBorderBottomLeftRadius: "26px",
			WebkitBorderBottomRightRadius: "26px",
		}
		return (
			<button
				type="submit"
				className="round"
				style={buttonStyle}
				>
				{this.props.title}
			</button>
		);
	}
});

module.exports = SubmitButtonView;