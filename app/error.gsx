package app

func Page() Node {
	return <main class="message-shell">
		<span class="overline error-label">500 · Studio error</span>
		<h1>Something broke</h1>
		<p>
			The app hit an unexpected error while rendering the current page.
		</p>
		<a href="/" data-gosx-link class="button primary">Return to Studio</a>
	</main>
}
