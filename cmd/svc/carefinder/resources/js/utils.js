export function	getQueryParams(): any {
	var params = {};
	var query = window.location.search.substring(1);
	if(query.length > 1) {
		var vars = query.split('&');
		for (var i = 0; i < vars.length; i++) {
				var pair = vars[i].split('=');
				var key = decodeURIComponent(pair[0]) 
				if(!params[key]){
					params[key] = []
				}
				params[key].push(decodeURIComponent(pair[1]))
		}
	}
	return params;
}