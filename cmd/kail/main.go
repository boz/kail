package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/boz/kail"
	"github.com/boz/kcache/util"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	flagNs    = kingpin.Flag("ns", "namespace").Strings()
	flagPod   = kingpin.Flag("pod", "pod").Strings()
	flagSvc   = kingpin.Flag("svc", "service").Strings()
	flagNode  = kingpin.Flag("node", "node").Strings()
	flagLabel = kingpin.Flag("label", "label").PlaceHolder("NAME=VALUE").Strings()

	flagDryRun = kingpin.Flag("dry-run", "print matching pods and exit").Bool()
)

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.CommandLine.Help = "Tail for kubernetes pods"
	kingpin.Parse()

	cs := kubeClient()

	dsb := kail.NewDSBuilder()

	if flagNs != nil {
		dsb = dsb.WithNamespace(*flagNs...)
	}

	ds := createDS(cs, dsb)

	listPods(ds)

}

func kubeClient() kubernetes.Interface {
	cs, _, err := util.KubeClient()
	kingpin.FatalIfError(err, "Error configuring kubernetes connection")

	_, err = cs.CoreV1().Namespaces().List(metav1.ListOptions{})
	kingpin.FatalIfError(err, "Can't connnect to kubernetes")

	return cs
}

func createDS(cs kubernetes.Interface, dsb kail.DSBuilder) kail.DS {
	ctx := context.Background()
	ds, err := dsb.Create(ctx, cs)
	kingpin.FatalIfError(err, "Error creating datasource")

	select {
	case <-ds.Ready():
	case <-ds.Done():
		kingpin.Fatalf("Unable to initialize data source")
	}
	return ds
}

func listPods(ds kail.DS) {
	pods, err := ds.Pods().Cache().List()
	kingpin.FatalIfError(err, "Error fetching pods")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintln(w, "NAMESPACE\tNAME\tNODE")

	for _, pod := range pods {
		fmt.Fprintf(w, "%v\t%v\t%v\n", pod.GetNamespace(), pod.GetName(), pod.Spec.NodeName)
	}

	w.Flush()
}
