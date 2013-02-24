Package rest is a RESTful web-service framework. It make struct method to http.Handler automatically.

Define a service struct like this:

	type RESTService struct {
		Service `root:"/root"`

		Hello    Processor `path:"/hello/(.*?)/to/(.*?)" method:"GET"`
		PostConv Processor `path:"/conversation" func:"PostConversation" method:"POST"`
		Conv     Processor `path:"/conversation/([0-9]+)" func:"GetConversation" method:"GET"`
	}

	func (s RESTService) Hello_(host, guest string) string {
		return "hello from " + host + " to " + guest
	}

	func (s RESTService) PostConversation(post string) string {
		path, _ := s.Conv.Path(1)
		s.RedirectTo(path)
		return "just post: " + post
	}

	func (s RESTService) GetConversation(id int) string {
		return fmt.Sprintf("get post id %d", id)
	}

The field tag of RESTService configure the parameters of processor, like method, path, or function which 
will process the request.

The path of processor can capture arguments, which will pass to process function by order in path. Arguments
type can be string or int, or any type which kind is string or int. 

The default name of processor is the name of field postfix with "_", like Hello processor correspond Hello_ method.

Get the http.Handler from RESTService:

	handler, err := rest.Init(new(RESTService))
	http.ListenAndServe("127.0.0.1:8080", handler)

Full document: http://godoc.org/github.com/googollee/go-rest