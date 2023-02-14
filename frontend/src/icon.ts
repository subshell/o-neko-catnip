import { mdiLoading, mdiOpenInNew } from '@mdi/js';

const supportedIcons: {[name: string]: string} = {
	mdiLoading,
	mdiOpenInNew
};

document.querySelectorAll("svg[data-icon]").forEach(item => {
	let iconName: string = item.getAttribute("data-icon") || "";
	if (!iconName || !supportedIcons[iconName]) {
		return;
	}
	setIcon(item, supportedIcons[iconName]);
});

function setIcon(container: Element, icon: string) {
	container.setAttribute("viewBox", "0 0 24 24");
	container.setAttribute("style", "display: inline-block; width: 1.5em;")
	container.innerHTML = `<path d="${icon}" style="fill:currentColor"></path>`;
}
