package checks

/*
- Data will be uploaded at set intervals, if it fails, it waits and retries in the next set interval, continuing to
add data to the queue.
- Individual checks are setup, such as ICMP to a target server, checking port uptimes, speedtests, mtr, twamp
- All checks are

Checks will have a type, target, result, interval, count

Mtr: target

*/

type CheckData struct {
	Type      string      `json:"type"`
	Target    string      `json:"address,omitempty"`
	ID        string      `json:"id"`
	Duration  int         `json:"interval,omitempty'"`
	Count     int         `json:"count,omitempty"`
	Triggered bool        `json:"triggered"`
	ToRemove  bool        `json:"toRemove"`
	Result    interface{} `json:"result"`
	Server    bool        `json:"server"`
}

/*func init() {
	for {
		buffer <- NewMtrCheck()
		buffer <- NewOtherCheck()
		wg := sync.WaitGroup{}
		for i := 0; i < runtime.NumCPU(); i++ {
			wg.Add(1)
			go func() {
				for {
					select {
					case next <- buffer:
						current.Run(CheckData{})
						done <- current.Results()
					default:
						wg.Done()
						break
					}
				}
			}()
		}
		wg.Wait()
		// Print all results once all done
	}
}*/

/**

Agent requests "config w/ checks" (eg. mtr:1.1.1.1 speedtest, blah)
Agent config will contain

CheckData:
- Type of Check
- Check Target (depending on type of check, IP, etc.) omitempty
- Check interval & count (omitempty)
- Result (format will vary depending on the check)

- Check data result will be individual structs based on check

*/
