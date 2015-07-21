/* @flow */

var React = require("react");
var Header = require("./Header.js")
var SectionedProgressBar = require("./SectionedProgressBar.js")
var Spinner = require("spin.js");

var SpinnerView = React.createClass({displayName: "SpinnerView",
	propTypes: {
		busy: React.PropTypes.bool.isRequired,
	},
	componentDidMount: function() {
		// NOTE: to adjust this, go to https://fgnass.github.io/spin.js/, and copy the `opts` var from that page
		var opts = {
		  lines: 11 // The number of lines to draw
		, length: 10 // The length of each line
		, width: 4 // The line thickness
		, radius: 9 // The radius of the inner circle
		, scale: 0.75 // Scales overall size of the spinner
		, corners: 1 // Corner roundness (0..1)
		, color: '#FFF' // #rgb or #rrggbb or array of colors
		, opacity: 0.25 // Opacity of the lines
		, rotate: 0 // The rotation offset
		, direction: 1 // 1: clockwise, -1: counterclockwise
		, speed: 1 // Rounds per second
		, trail: 60 // Afterglow percentage
		, fps: 20 // Frames per second when using setTimeout() as a fallback for CSS
		, zIndex: 2e9 // The z-index (defaults to 2000000000)
		, className: 'spinner' // The CSS class to assign to the spinner
		, top: '50%' // Top position relative to parent
		, left: '50%' // Left position relative to parent
		, shadow: false // Whether to render a shadow
		, hwaccel: false // Whether to use hardware acceleration
		, position: 'absolute' // Element positioning
		}
		var target = React.findDOMNode(this.refs.spinner);
		var spinner = new Spinner(opts).spin(target);
	},
	render: function(): any {

		return (
			<div style={{
				position: "fixed",
				width: "100%",
				height: "100%",
				backgroundColor: "rgba(0,0,0,0.1)",
				verticalAlign: "middle",
				zIndex: "9999",
				display: (this.props.busy ? "" : "none"),
			}}>
				<div style={{
					width: "100%",
					height: "100%",
					backgroundColor: "rgba(0,0,0,0.7)",
					margin: "auto",
					display: "inline-block",
					position: "absolute",
					left: "50%",
					top: "50%",
					transform: "translate(-50%,-50%)",
					WebkitTransform: "translate(-50%,-50%)",
					borderRadius: "10px",
				}}>
					<div ref="spinner"></div>
				</div>
			</div>
		);
	}
});

var maxContentWidth = 414 // iPhone 6 Plus width
var contentPaneStyle = {
	maxWidth: maxContentWidth,
	marginLeft: "auto",
	marginRight: "auto",
}
var contentPaneContainerStyle = {
	paddingLeft: "16px",
	paddingRight: "16px",
}

var ContentContainer = React.createClass({displayName: "ContentContainer",
	propTypes: {
		busy: React.PropTypes.bool.isRequired,
		content: React.PropTypes.node.isRequired,
		showSectionedProgressBar: React.PropTypes.bool.isRequired,
		numSections: React.PropTypes.number,
		currentSectionIndex: React.PropTypes.number,
	},
	render: function(): any {
		var numSections: number = (this.props.numSections ? this.props.numSections : 0)
		return (
			<div>
				<SpinnerView busy={this.props.busy}/>
				<Header />
				{this.props.showSectionedProgressBar ? 
					<SectionedProgressBar 
						currentSectionIndex={this.props.currentSectionIndex ? this.props.currentSectionIndex : 0}
						numSections={numSections} 
						maxWidth={maxContentWidth} />
				: null}
				<div style={contentPaneStyle}>
					<div style={contentPaneContainerStyle}>
						{this.props.content}
					</div>
				</div>
			</div>
		);
	}
});

module.exports = ContentContainer;