/* @flow */

var React = require("react");
var Utils = require("../../libs/utils.js");

var Analytics = require("../../libs/analytics.js");
var AnalyticsScreenName = "confirmation"
var Constants = require("./Constants.js");

var Header = React.createClass({displayName: "Header",
	render: function(): any {
		return (
			<div id="header"
				style={{
					zIndex: "2",
					position: "relative",
					width: "100%",
					textAlign: "center",
					verticalAlign: "middle",
					MozBoxShadow: "0px 1px 1px 1px rgba(0, 0, 0, 0.1)",
					WebkitBoxShadow: "0px 1px 1px 1px rgba(0, 0, 0, 0.1)",
					boxShadow: "0px 1px 1px 1px rgba(0, 0, 0, 0.1)",
				}}>
				<div style={{
						paddingTop: "10px",
						paddingBottom: "8px",
					}}>
					<a href="https://www.sprucehealth.com" onClick={function (e: any) {
						// Warning: this is a synchronous request
						Analytics.record("spruce_logo_clicked", {"app_type": Constants.AnalyticsAppType, "screen_id": AnalyticsScreenName}, true)
					}}>
						<img 
							src={Utils.staticURL("/img/pc/logo@2x.png")}
							style={{
								width: "123px",
								height: "33px",
							}} />
					</a>
				</div>
			</div>
		);
	}
});

module.exports = Header;