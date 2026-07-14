package app

func Page() Node {
	return <main class="message-shell">
		<span class="overline">404 · Missing surface</span>
		<h1>Page not found</h1>
		<p>
			The requested Studio route is not part of the current scaffold.
		</p>
		<a href="/" data-gosx-link class="button primary">Return to Studio</a>
	</main>
}
