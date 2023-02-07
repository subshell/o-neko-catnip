interface StatusResponse {
	deploymentReady: boolean;
	redirectUrl: string;
	isError: boolean;
	errorMessage: string;
}

const deploymentUrl = new URLSearchParams(location.search).get("redirectTo") || "";

function checkDeploymentStatus() {
	if (deploymentUrl === "") {
		return;
	}

	fetch(`/api/status?deploymentUrl=${deploymentUrl}`, {method: "GET"})
		.then(response => {
				if (response.status > 500) {
					console.log("failed to get deployment status");
					retry();
					return null;
				}
				return response.json()
			}
		).then((response: StatusResponse) => {
			if (response == null) {
				return;
			}

			if (response.isError) {
				console.log("failed to check deployment status: " + response.errorMessage);
				retry();
				return;
			}

			if (response.deploymentReady) {
				redirectToDeployment();
				return;
			}

			retry();
		}
	);
}

function retry() {
	setTimeout(() => checkDeploymentStatus(), 1000);
}

function redirectToDeployment() {
	window.location.replace(deploymentUrl);
}

checkDeploymentStatus();

export {};
