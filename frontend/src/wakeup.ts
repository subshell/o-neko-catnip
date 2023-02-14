import Alpine from "alpinejs";

type DeploymentStatus = "Pending" | "Ready" | "Error";

interface StatusResponse {
	deploymentStatus: DeploymentStatus;
	redirectUrl: string;
	errorMessage: string;
}

interface WakeupPageComponent {
	currentStatus: StatusResponse;
	deploymentUrl: string;
	checkDeploymentStatus: () => void;
	redirectAfterDelay: () => void;
	redirectToDeployment: () => void;
	retry: () => void;
}

const component: WakeupPageComponent = {
	deploymentUrl: new URLSearchParams(location.search).get("redirectTo") || "",
	currentStatus: {
		deploymentStatus: "Pending",
		redirectUrl: "",
		errorMessage: ""
	},
	checkDeploymentStatus() {
		if (this.deploymentUrl === "") {
			return;
		}

		fetch(`/api/status?deploymentUrl=${this.deploymentUrl}`, {method: "GET"})
			.then(response => {
					if (response.status > 500) {
						console.log("failed to get deployment status");
						this.retry();
						return null;
					}
					return response.json()
				}
			).then((response: StatusResponse) => {
				if (response == null) {
					return;
				}

				this.currentStatus = response;

				if (response.deploymentStatus == "Error") {
					console.log("failed to check deployment status: " + response.errorMessage);
					this.retry();
					return;
				}

				if (response.deploymentStatus == "Ready") {
					this.redirectAfterDelay();
					return;
				}

				this.retry();
			}
		);
	},
	retry() {
		setTimeout(() => this.checkDeploymentStatus(), 1000);
	},
	redirectAfterDelay() {
		setTimeout(() => this.redirectToDeployment(), 6000);
	},
	redirectToDeployment() {
		window.location.replace(this.deploymentUrl);
	}
}

Alpine.data("wakeup", () => component)
Alpine.start();
