# SNIPPETS

https://github.com/xh3b4sd/matic/blob/master/collector/collector.go

func (gcg *GoClientCollector) GenerateClient(wd string) error {
	// Create task context.
	ctx := &Ctx{
		WorkingDir: wd,
	}

	// Create a new queue.
	q := taskqPkg.NewQueue(ctx)

	// Run tasks.
	err := q.RunTasks(
		taskqPkg.InSeries(
			SourceCodeTask,
			PackageImportTask,
			ServerNameTask,
			ServeStmtTask,
			// find middlewares for each route
			// find possible responses for each route
		),
	)

	if err != nil {
		return Mask(err)
	}

	return nil
}

