/* @flow */

import * as React from "react";
import * as ReactDOM from "react-dom";
import * as API from "./api";
import * as Utils from "./utils";

window.React = React;
window.ReactDOM = ReactDOM;

window.TextMe = React.createClass({displayName: "TextMe",
		propTypes: {
			doctorID: React.PropTypes.string.isRequired,
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
			API.textDownloadLink(this.props.doctorID, this.state.number, function(success, data, error) {
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
							border: "2px solid #0E94B0",
							borderRadius: 4,
							padding: "15px",
							fontFamily: "'MuseoSans-300', Helvetica, Arial, sans-serif",
							fontSize: "16px",
							lineHeight: "19px",
							marginTop: "20px",
						}}>
						Download link sent to {this.state.number}
					</div>
				);
			}
			return (
				<div
					style = {{
						marginTop: "20px",
					}}
				>
					<form id="cftextlink" onSubmit={this.handleSubmit}>
						<input
							type = "text"
							required = {true}
							value = {this.state.number}
							onChange = {this.handleChangeNumber}
							placeholder = "Enter Your Mobile Phone #"
							size = {28}
							style = {{
								border: "2px solid #0E94B0",
								borderRight: 0,
								borderTopLeftRadius: 4,
								borderBottomLeftRadius: 4,
								paddingLeft: "15px",
								paddingRight: "15px",
								marginRight: "0px",
								height: "44px",
								fontFamily: "'MuseoSans-300', Helvetica, Arial, sans-serif",
								fontSize: "14px",
								lineHeight: "17px",
							}} />
						<button
							type = "submit"
							style = {{
								verticalAlign: "top",
								border: "2px solid #0E94B0",
								borderLeft: 0,
								borderTopRightRadius: 4,
								borderBottomRightRadius: 4,
								paddingRight: "15px",
								paddingLeft: "15px",
								marginLeft: "0px",
								height: "50px",
								color: "#fff",
								backgroundColor: "#00CECF",
								fontFamily: "'MuseoSans-700', Helvetica, Arial, sans-serif",
								fontSize: "14px",
								lineHeight: "17px",
							}}>
							GET STARTED
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
	});
