/* @flow */

var React = require("react");

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
					<a href="https://www.sprucehealth.com">
						<img 
							src="https://dlzz6qy5jmbag.cloudfront.net/web/11/img/logo-small.png"
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