package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	logutil "github.com/boz/go-logutil"
	logutil_logrus "github.com/boz/go-logutil/logrus"
	"github.com/boz/kail"
	"github.com/boz/kcache/nsname"
	"github.com/boz/kcache/util"
	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	flagNs    = kingpin.Flag("ns", "namespace").PlaceHolder("NAMESPACE-NAME").Strings()
	flagPod   = kingpin.Flag("pod", "pod").PlaceHolder("POD-NAME").Strings()
	flagSvc   = kingpin.Flag("svc", "service").PlaceHolder("SERVICE-NAME").Strings()
	flagRc    = kingpin.Flag("rc", "replication controller").PlaceHolder("RC-NAME").Strings()
	flagNode  = kingpin.Flag("node", "node").PlaceHolder("NODE-NAME").Strings()
	flagLabel = kingpin.Flag("label", "label").PlaceHolder("NAME=VALUE").Strings()

	flagDryRun = kingpin.Flag("dry-run", "print matching pods and exit").
			Default("false").
			Bool()

	flagLogFile = kingpin.Flag("log-file", "log file output").
			Default("/dev/stderr").
			OpenFile(os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

	flagLogLevel = kingpin.Flag("log-level", "log level").
			Default("error").
			Enum("debug", "info", "warn", "error")
)

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.CommandLine.Help = "Tail for kubernetes pods"
	kingpin.Parse()

	log := createLog()

	cs := createKubeClient()

	dsb := createDSBuilder()

	ctx := logutil.NewContext(context.Background(), log)

	ds := createDS(ctx, cs, dsb)

	listPods(ds)

	if !*flagDryRun {
		streamLogs(ctx, cs, ds)
		return
	}

	ds.Shutdown()
	<-ds.Done()
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

func createDSBuilder() kail.DSBuilder {
	dsb := kail.NewDSBuilder()
	if flagNs != nil {
		dsb = dsb.WithNamespace(*flagNs...)
	}

	if ids := parseIds("pod", flagPod); len(ids) > 0 {
		dsb = dsb.WithPods(ids...)
	}

	if flagLabel != nil {
		labels := make(map[string]string)
		for _, val := range *flagLabel {
			parts := strings.Split(val, "=")
			switch len(parts) {
			case 2:
				labels[parts[0]] = parts[1]
			default:
				kingpin.Fatalf("Invalid label: '%v'", val)
			}
		}
		if len(labels) > 0 {
			dsb = dsb.WithLabels(labels)
		}
	}

	if ids := parseIds("service", flagSvc); len(ids) > 0 {
		dsb = dsb.WithService(ids...)
	}

	if flagNode != nil {
		dsb = dsb.WithNode(*flagNode...)
	}

	if ids := parseIds("rc", flagRc); len(ids) > 0 {
		dsb = dsb.WithRC(ids...)
	}

	return dsb
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

func streamLogs(ctx context.Context, cs kubernetes.Interface, ds kail.DS) {
	controller, err := kail.NewController(ctx, cs, ds.Pods())
	kingpin.FatalIfError(err, "Error creating controller")

	writer := kail.NewWriter(os.Stdout)

	for {
		select {
		case ev := <-controller.Events():
			writer.Print(ev)
		case <-controller.Done():
			return
		}
	}
}

func parseIds(name string, vals *[]string) []nsname.NSName {
	var ids []nsname.NSName

	if vals == nil {
		return ids
	}

	for _, val := range *vals {
		parts := strings.Split(val, "/")
		switch len(parts) {
		case 2:
			ids = append(ids, nsname.New(parts[0], parts[1]))
		case 1:
			ids = append(ids, nsname.New("", parts[0]))
		default:
			kingpin.Fatalf("Invalid %v name: '%v'", name, val)
		}
	}

	return ids
}
