function checkIfStillOneko() {
	fetch(location.href)
		.then(response => {
			if (response.headers.has("oneko-url-trigger")) {
				// still oneko
				console.log("still o-neko");
				setTimeout(() => checkIfStillOneko(), 1000);
			} else {
				console.log("not o-neko anymore. reloading");
				location.reload();
			}
		}).catch(err => {
			// let's try again
		console.error(err);
		setTimeout(() => checkIfStillOneko(), 1000);
	});
}

checkIfStillOneko();
