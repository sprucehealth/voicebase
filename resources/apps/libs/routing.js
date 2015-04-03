/* @flow */

module.exports = {
	RouterNavigateMixin: {
		navigate: function(path: string): void {
			if (path.indexOf(this.props.router.root) == 0) {
				path = path.substring(this.props.router.root.length, path.length);
			}
			this.props.router.navigate(path, {
				trigger : true
			});
		},
		onNavigate: function(e: any): void {
			e.preventDefault();
			var el = e.target;
			// Find the closest ancestor with an href
			while (typeof el != "undefined" && typeof el.pathname == "undefined") {
				el = el.parentNode;
			}
			this.navigate(el.pathname);
		}
	}
}
