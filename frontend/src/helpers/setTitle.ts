export function setTitle(title : undefined | string) {
	document.title = (typeof title === 'undefined' || title === '')
		? 'Fenster'
		: `${title} | Fenster`
}
