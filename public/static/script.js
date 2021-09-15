function checkIfDeploymentIsReady() {
	fetch(location.href, {method: 'HEAD'})
		.then(response => {
			if (response.headers.has("oneko-url-trigger")) {
				// still oneko
				console.log("still o-neko");
				setTimeout(() => checkIfDeploymentIsReady(), 1000);
			} else {
				console.log("not o-neko anymore. reloading");
				location.reload();
			}
		}).catch(err => {
			// let's try again
		console.error(err);
		setTimeout(() => checkIfDeploymentIsReady(), 1000);
	});
}

checkIfDeploymentIsReady();
