package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	logutil "github.com/boz/go-logutil"
	logutil_logrus "github.com/boz/go-logutil/logrus"
	"github.com/boz/kail"
	"github.com/boz/kcache/util"
	"github.com/sirupsen/logrus"
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

	flagLogFile = kingpin.Flag("log-file", "log file output").
			Default("/dev/stderr").
			OpenFile(os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

	flagLogLevel = kingpin.Flag("log-level", "log level").
			Default("warn").
			Enum("debug", "info", "warn", "error")
)

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.CommandLine.Help = "Tail for kubernetes pods"
	kingpin.Parse()

	log := createLog()

	cs := createKubeClient()

	dsb := kail.NewDSBuilder()

	if flagNs != nil {
		dsb = dsb.WithNamespace(*flagNs...)
	}

	ctx := logutil.NewContext(context.Background(), log)

	ds := createDS(ctx, cs, dsb)

	listPods(ds)

	controller, err := kail.NewController(ctx, cs, ds.Pods())
	kingpin.FatalIfError(err, "Error creating controller")

	for {
		select {
		case ev := <-controller.Events():
			fmt.Printf("%v/%v:%v\t", ev.Source().Namespace(), ev.Source().Name(), ev.Source().Container())
			fmt.Println(ev.Log())
		case <-controller.Done():
			return
		}
	}
}

func createLog() logutil.Log {
	lvl, err := logrus.ParseLevel(*flagLogLevel)
	kingpin.FatalIfError(err, "Invalid log level")

	parent := logrus.New()
	parent.Level = lvl
	parent.Out = *flagLogFile

	return logutil_logrus.New(parent)
}

func createKubeClient() kubernetes.Interface {
	cs, _, err := util.KubeClient()
	kingpin.FatalIfError(err, "Error configuring kubernetes connection")

	_, err = cs.CoreV1().Namespaces().List(metav1.ListOptions{})
	kingpin.FatalIfError(err, "Can't connnect to kubernetes")

	return cs
}

func createDS(ctx context.Context, cs kubernetes.Interface, dsb kail.DSBuilder) kail.DS {
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
