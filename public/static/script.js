function checkIfDeploymentIsReady() {
	fetch(location.href, {method: 'HEAD'})
		.then(response => {
			if (response.status > 500) {
				// apparently something is happening but the service is not available yet
				console.log("new deployment not available yet");
				setTimeout(() => checkIfDeploymentIsReady(), 1000);
			} else if (response.headers.has("oneko-catnip")) {
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
