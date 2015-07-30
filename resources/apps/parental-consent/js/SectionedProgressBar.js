/* @flow */

var React = require("react");
var Utils = require("../../libs/utils.js");

var SectionedProgressBar = React.createClass({displayName: "SectionedProgressBar",
	propTypes: {
		numSections: React.PropTypes.number.isRequired,
		currentSectionIndex: React.PropTypes.number.isRequired,
		maxWidth: React.PropTypes.number.isRequired,
	},
	render: function(): any {

		var numSections: number = this.props.numSections
		var currentSectionIndex: number = this.props.currentSectionIndex
		var sections = []
		var leftStyle = {
			borderLeftWidth: "2px",
			borderLeftColor: "white",
			borderLeftStyle: "solid",
			borderRightWidth: "1px",
			borderRightColor: "white",
			borderRightStyle: "solid",
		}
		var innerStyle = {
			borderLeftWidth: "1px",
			borderLeftColor: "white",
			borderLeftStyle: "solid",
			borderRightWidth: "1px",
			borderRightColor: "white",
			borderRightStyle: "solid",
		}
		var rightStyle = {
			borderLeftWidth: "1px",
			borderLeftColor: "white",
			borderLeftStyle: "solid",
			borderRightWidth: "2px",
			borderRightColor: "white",
			borderRightStyle: "solid",
		}
		var commonStyle = {
			width: Math.round(1.0 / numSections * 100) + "%",
			height: "100%",
			backgroundColor: "#e2ecef",
		}
		var filledStyle = {
			backgroundColor: "#1fa6c4",
		}
		for (var i = 0; i < numSections; ++i) {
			var style = commonStyle
			if (i == 0) {
				style = Utils.mergeProperties(style, leftStyle)
			} else if (i + 1 == numSections) {
				style = Utils.mergeProperties(style, rightStyle)
			} else {
				style = Utils.mergeProperties(style, innerStyle)
			}

			if (i <= currentSectionIndex) {
				style = Utils.mergeProperties(style, filledStyle)
			}

			sections.push(
				<div style={style} key={i}></div>
				)
		}
		return (
			<div style={{
					height: "4px",
					backgroundColor: "#f5f9fa",
					zIndex: "1",
					position: "relative",
				}}>
				<div className="flexBox" style={{
						display: "flex",
						maxWidth: this.props.maxWidth,
						height: "100%",
						backgroundColor: "white",
						marginLeft: "auto",
						marginRight: "auto",
					}}>
					{sections}
				</div>
			</div>
		);
	}
});

module.exports = SectionedProgressBar;