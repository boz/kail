package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	logutil "github.com/boz/go-logutil"
	logutil_logrus "github.com/boz/go-logutil/logrus"
	"github.com/boz/kail"
	"github.com/boz/kcache/nsname"
	"github.com/boz/kcache/util"
	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	flagIgnore = kingpin.Flag("ignore", "ignore selector").PlaceHolder("SELECTOR").Default("kail.ignore=true").Strings()

	flagLabel      = kingpin.Flag("label", "label").Short('l').PlaceHolder("SELECTOR").Strings()
	flagPod        = kingpin.Flag("pod", "pod").Short('p').PlaceHolder("NAME").Strings()
	flagNs         = kingpin.Flag("ns", "namespace").Short('n').PlaceHolder("NAME").Strings()
	flagSvc        = kingpin.Flag("svc", "service").PlaceHolder("NAME").Strings()
	flagRc         = kingpin.Flag("rc", "replication controller").PlaceHolder("NAME").Strings()
	flagRs         = kingpin.Flag("rs", "replica set").PlaceHolder("NAME").Strings()
	flagDs         = kingpin.Flag("ds", "daemonset").PlaceHolder("NAME").Strings()
	flagDeployment = kingpin.Flag("deploy", "deployment").Short('d').PlaceHolder("NAME").Strings()
	flagNode       = kingpin.Flag("node", "node").PlaceHolder("NAME").Strings()
	flagIng        = kingpin.Flag("ing", "ingress").PlaceHolder("NAME").Strings()

	flagContext = kingpin.Flag("context", "kubernetes context").PlaceHolder("CONTEXT-NAME").String()

	flagContainers = kingpin.Flag("containers", "containers").Short('c').PlaceHolder("NAME").Strings()

	flagDryRun = kingpin.Flag("dry-run", "print matching pods and exit").
			Default("false").
			Bool()

	flagLogFile = kingpin.Flag("log-file", "log file output").
			Default("/dev/stderr").
			OpenFile(os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

	flagLogLevel = kingpin.Flag("log-level", "log level").
			Default("error").
			Enum("debug", "info", "warn", "error")

	flagSince = kingpin.Flag("since", "Display logs generated since given duration, like 5s, 2m, 1.5h or 2h45m. Defaults to 1s.").
			PlaceHolder("DURATION").
			Default("1s").
			Duration()

	flagGlogV = kingpin.Flag("glog-v", "glog -v value").
			Default("0").
			String()

	flagGlogVmodule = kingpin.Flag("glog-vmodule", "glog -vmodule flag").
			Default("").
			String()
)

func main() {

	// XXX: hack to make kubectl run work
	if os.Args[len(os.Args)-1] == "" {
		os.Args = os.Args[0 : len(os.Args)-1]
	}

	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.CommandLine.Help = "Tail for kubernetes pods"
	kingpin.Parse()

	log := createLog()

	cs, rc := createKubeClient()

	dsb := createDSBuilder()

	ctx := logutil.NewContext(context.Background(), log)

	ctx, cancel := context.WithCancel(ctx)

	sigch := watchSignals(ctx, cancel)

	ds := createDS(ctx, cs, dsb)

	filter := kail.NewContainerFilter(*flagContainers)

	if *flagDryRun {

		listPods(ds, filter)

	} else {

		streamLogs(createController(ctx, cs, rc, ds, filter))

	}

	cancel()
	<-ds.Done()
	<-sigch
}

func watchSignals(ctx context.Context, cancel context.CancelFunc) <-chan struct{} {
	donech := make(chan struct{})
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		defer close(donech)
		defer signal.Stop(sigch)
		select {
		case <-ctx.Done():
		case <-sigch:
			cancel()
		}
	}()
	return donech
}

func createLog() logutil.Log {
	lvl, err := logrus.ParseLevel(*flagLogLevel)
	kingpin.FatalIfError(err, "Invalid log level")

	parent := logrus.New()
	parent.Level = lvl
	parent.Out = *flagLogFile

	// XXX: fucking glog.
	os.Args = []string{os.Args[0],
		"-logtostderr=true",
		"-v=" + *flagGlogV,
		"-vmodule=" + *flagGlogVmodule,
	}
	flag.Parse()

	return logutil_logrus.New(parent).WithComponent("kail.main")
}

func createKubeClient() (kubernetes.Interface, *rest.Config) {
	overrides := &clientcmd.ConfigOverrides{}

	if flagContext != nil {
		overrides.CurrentContext = *flagContext
	}

	cs, rc, err := util.KubeClient(overrides)
	kingpin.FatalIfError(err, "Error configuring kubernetes connection")

	_, err = cs.CoreV1().Namespaces().List(metav1.ListOptions{})
	kingpin.FatalIfError(err, "Can't connnect to kubernetes")

	return cs, rc
}

func createDSBuilder() kail.DSBuilder {
	dsb := kail.NewDSBuilder()

	if selectors := parseLabels("ignore", *flagIgnore); len(selectors) > 0 {
		dsb = dsb.WithIgnore(selectors...)
	}

	if selectors := parseLabels("selector", *flagLabel); len(selectors) > 0 {
		dsb = dsb.WithSelectors(selectors...)
	}

	if ids := parseIds("pod", *flagPod); len(ids) > 0 {
		dsb = dsb.WithPods(ids...)
	}

	if len(*flagNs) > 0 {
		dsb = dsb.WithNamespace(*flagNs...)
	}

	if ids := parseIds("service", *flagSvc); len(ids) > 0 {
		dsb = dsb.WithService(ids...)
	}

	if len(*flagNode) > 0 {
		dsb = dsb.WithNode(*flagNode...)
	}

	if ids := parseIds("rc", *flagRc); len(ids) > 0 {
		dsb = dsb.WithRC(ids...)
	}

	if ids := parseIds("rs", *flagRs); len(ids) > 0 {
		dsb = dsb.WithRS(ids...)
	}

	if ids := parseIds("ds", *flagDs); len(ids) > 0 {
		dsb = dsb.WithDS(ids...)
	}

	if ids := parseIds("deploy", *flagDeployment); len(ids) > 0 {
		dsb = dsb.WithDeployment(ids...)
	}

	if ids := parseIds("ing", *flagIng); len(ids) > 0 {
		dsb = dsb.WithIngress(ids...)
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

func listPods(ds kail.DS, filter kail.ContainerFilter) {
	pods, err := ds.Pods().Cache().List()
	kingpin.FatalIfError(err, "Error fetching pods")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintln(w, "NAMESPACE\tNAME\tCONTAINER\tNODE")

	for _, pod := range pods {
		_, sources := kail.SourcesForPod(filter, pod)
		for _, source := range sources {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", source.Namespace(), source.Name(), source.Container(), source.Node())
		}
	}

	w.Flush()
}

func createController(
	ctx context.Context, cs kubernetes.Interface, rc *rest.Config, ds kail.DS, filter kail.ContainerFilter) kail.Controller {

	controller, err := kail.NewController(ctx, cs, rc, ds.Pods(), filter, *flagSince)
	kingpin.FatalIfError(err, "Error creating controller")

	return controller
}

func streamLogs(controller kail.Controller) {

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

func parseLabels(name string, vals []string) []labels.Selector {
	var selectors []labels.Selector
	for _, val := range vals {
		selector, err := labels.Parse(val)
		kingpin.FatalIfError(err, "invalid %v labels expression: '%v'", name, val)
		selectors = append(selectors, selector)
	}
	return selectors
}

func parseIds(name string, vals []string) []nsname.NSName {
	var ids []nsname.NSName

	for _, val := range vals {
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
