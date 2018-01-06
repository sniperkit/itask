package xtask

// ContinueWithHandler
type ContinueWithHandler func(TaskResult)

// ContinueWithHandler
type ContinueWithFunc func(TaskResult)

// ContinueWithHandler
type ContinueWithTask func(TaskResult)

// TaskResult
type TaskResult struct {
	done   chan bool
	Result interface{}
	Error  error
}

/*
// ref. https://github.com/cxww107/asyncwork#get-result-from-completed-tasks
func (task *Task) TaskResultChan() *Task {
	done := make(*TaskResult chan)
	for result := range resultChannel {
		switch result.(type) {
		case error:
			log.Println("Received error")
			cancel()
			return
		case log:
			fmt.Println("Here is a string:", result.(string))
		case int:
			log.Println("Here is an integer:", result.(int))
		default:
			log.Println("Some unknown type ")
		}
	}
}

func (task *Task) TaskResultFormat(res *TaskResult) *Task {
	for result := range resultChannel {
		switch result.(type) {
		case error:
			log.Println("Received error")
			cancel()
			return
		case string:
			log.Println("Here is a string:", result.(string))
		case int:
			log.Println("Here is an integer:", result.(int))
		default:
			log.Println("Some unknown type ")
		}
	}
}

func (task *Task) AggregateResults(res *TaskResult) *Task {
	for result := range resultChannel {
		switch result.(type) {
		case error:
			log.Println("Received error")
			cancel()
			return
		case string:
			log.Println("Here is a string:", result.(string))
		case int:
			log.Println("Here is an integer:", result.(int))
		default:
			log.Println("Some unknown type ")
		}
	}
}

*/
