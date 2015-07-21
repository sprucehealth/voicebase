/* @flow */

var React = require("react/addons");


var TitleView = React.createClass({displayName: "TitleView",
	propTypes: {
		title: React.PropTypes.string.isRequired,
		subtitle: React.PropTypes.string.isRequired,
		text: React.PropTypes.string,
	},
	render: function(): any {
		return (
			<div>
				<div style={{
						marginTop: "16px",
						textAlign: "center",
						color: "#8B9FA7",
					}}>{this.props.title}</div>
				<div style={{
						marginTop: "6px",
						fontFamily: "MuseoSans-500",
						fontSize: "20px",
						lineHeight: "24px",
						textAlign: "center",
					}}>{this.props.subtitle}</div>
				{(this.props.text && this.props.text.length) ?
					<div style={{
						marginTop: "12px",
						fontFamily: "MuseoSans-300",
						fontSize: "14px",
						lineHeight: "17px",
						textAlign: "center",
						color: "#728289",
					}}>{this.props.text}</div>
				: null}
			</div>
		);
	}
});

module.exports = TitleView;
