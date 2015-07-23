/* @flow */

var API = require("./api.js");
var React = require('react/addons');
var Utils = require("../../libs/utils.js");

module.exports = {
	Component: React.createClass({displayName: "TextMe",
		propTypes: {
			code: React.PropTypes.string.isRequired,
		},
		getInitialState: function() {
			return {
				number: "",
				error: "",
				done: false
			};
		},
		handleChangeNumber: function(e: any) {
			e.preventDefault();
			this.setState({number: e.target.value});
		},
		handleSubmit: function(e: Event) {
			e.preventDefault();
			console.log(this.props.code, Utils.getQueryParams())
			API.textDownloadLink(this.props.code, Utils.getQueryParams(), this.state.number, function(success, data, error) {
				if (this.isMounted()) {
					if (!success) {
						this.setState({error: error.message});
						return;
					}
					if (data.success) {
						this.setState({done: true});
					} else {
						this.setState({error: data.error});
					}
				}
			}.bind(this));
		},
		render: function(): any {
			if (this.state.done) {
				return (
					<div
						style = {{
							backgroundColor: "#fff",
							color: "#1E333A",
							border: "3px solid #d8e8ed",
							borderRadius: 55,
							padding: "15px",
							fontFamily: "'MuseoSans-300', Helvetica, Arial, sans-serif",
							fontSize: "16px",
							lineHeight: "19px",
						}}>
						<span
							className = "glyphicon glyphicon-ok"
							style={{
								color: "#00CECF",
								marginRight: 10,
							}}></span>
						Download link sent to {this.state.number}
					</div>
				);
			}
			return (
				<div>
					<form onSubmit={this.handleSubmit} className="form">
						<input
							type = "text"
							required = {true}
							value = {this.state.number}
							onChange = {this.handleChangeNumber}
							placeholder = "Enter phone number"
							style = {{
								border: "3px solid #d8e8ed",
								borderRight: 0,
								borderTopLeftRadius: 55,
								borderBottomLeftRadius: 55,
								padding: "15px 15px 15px 25px",
								fontFamily: "'MuseoSans-300', Helvetica, Arial, sans-serif",
								fontSize: 16,
								lineHeight: "19px",
							}} />
						<button
							type = "submit"
							style = {{
								border: "3px solid #d8e8ed",
								borderLeft: 0,
								borderTopRightRadius: 55,
								borderBottomRightRadius: 55,
								padding: "16px 26px 16px 16px",
								color: "#fff",
								backgroundColor: "#00CECF",
								fontFamily: "'MuseoSans-700', Helvetica, Arial, sans-serif",
								fontSize: 14,
								lineHeight: "17px",
							}}>
							TEXT DOWNLOAD LINK
						</button>
					</form>
					{this.state.error ?
						<div
							style={{
								color: "red",
								padding: "8px 15px",
								fontFamily: "'MuseoSans-700', Helvetica, Arial, sans-serif",
								fontSize: 14,
								lineHeight: "17px",
							}}>{this.state.error}</div> : null}
				</div>
			);
		}
	})
};
