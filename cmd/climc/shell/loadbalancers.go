package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	R(&options.LoadbalancerCreateOptions{}, "lb-create", "Create lb", func(s *mcclient.ClientSession, opts *options.LoadbalancerCreateOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		lb, err := modules.Loadbalancers.Create(s, params)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})
	R(&options.LoadbalancerGetOptions{}, "lb-show", "Show lb", func(s *mcclient.ClientSession, opts *options.LoadbalancerGetOptions) error {
		lb, err := modules.Loadbalancers.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})
	R(&options.LoadbalancerListOptions{}, "lb-list", "List lbs", func(s *mcclient.ClientSession, opts *options.LoadbalancerListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Loadbalancers.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Loadbalancers.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerUpdateOptions{}, "lb-update", "Update lb", func(s *mcclient.ClientSession, opts *options.LoadbalancerUpdateOptions) error {
		params, err := options.StructToParams(opts)
		lb, err := modules.Loadbalancers.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})
	R(&options.LoadbalancerDeleteOptions{}, "lb-delete", "Show lb", func(s *mcclient.ClientSession, opts *options.LoadbalancerDeleteOptions) error {
		lb, err := modules.Loadbalancers.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})
	R(&options.LoadbalancerActionStatusOptions{}, "lb-status", "Change lb status", func(s *mcclient.ClientSession, opts *options.LoadbalancerActionStatusOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		lb, err := modules.Loadbalancers.PerformAction(s, opts.ID, "status", params)
		if err != nil {
			return err
		}
		printObject(lb)
		return nil
	})
}