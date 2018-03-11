package main

type model struct {
	BuildSlug   string   `json:"build_slug"`
	TestResults []result `json:"test_results"`
}

type result struct {
	Path    string `json:"path"`
	Content []byte `json:"content"`
}
