package model

import (
	"fmt"
	"log"
	"strings"

	"github.com/deis/router/utils"
	modelerUtility "github.com/deis/router/utils/modeler"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/labels"
)

const (
	prefix               string = "router.deis.io"
	modelerFieldTag      string = "key"
	modelerConstraintTag string = "constraint"
)

var (
	namespace        = utils.GetOpt("POD_NAMESPACE", "default")
	modeler          = modelerUtility.NewModeler(prefix, modelerFieldTag, modelerConstraintTag, true)
	servicesSelector labels.Selector
)

func init() {
	var err error
	servicesSelector, err = labels.Parse(fmt.Sprintf("%s/routable==true", prefix))
	if err != nil {
		log.Fatal(err)
	}
}

// RouterConfig is the primary type used to encapsulate all router configuration.
type RouterConfig struct {
	WorkerProcesses          string      `key:"workerProcesses" constraint:"^(auto|[1-9]\\d*)$"`
	MaxWorkerConnections     string      `key:"maxWorkerConnections" constraint:"^[1-9]\\d*$"`
	TrafficStatusZoneSize    string      `key:"trafficStatusZoneSize" constraint:"^[1-9]\\d*[kKmM]?$"`
	DefaultTimeout           string      `key:"defaultTimeout" constraint:"^[1-9]\\d*(ms|[smhdwMy])?$"`
	ServerNameHashMaxSize    string      `key:"serverNameHashMaxSize" constraint:"^[1-9]\\d*[kKmM]?$"`
	ServerNameHashBucketSize string      `key:"serverNameHashBucketSize" constraint:"^[1-9]\\d*[kKmM]?$"`
	GzipConfig               *GzipConfig `key:"gzip"`
	BodySize                 string      `key:"bodySize" constraint:"^[1-9]\\d*[kKmM]?$"`
	ProxyRealIPCIDRs         []string    `key:"proxyRealIpCidrs" constraint:"^((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\\/([0-9]|[1-2][0-9]|3[0-2]))?(\\s*,\\s*)?)+$"`
	ErrorLogLevel            string      `key:"errorLogLevel" constraint:"^(info|notice|warn|error|crit|alert|emerg)$"`
	PlatformDomain           string      `key:"platformDomain" constraint:"(?i)^([a-z0-9]+(-[a-z0-9]+)*\\.)+[a-z]{2,}$"`
	UseProxyProtocol         bool        `key:"useProxyProtocol" constraint:"(?i)^(true|false)$"`
	EnforceWhitelists        bool        `key:"enforceWhitelists" constraint:"(?i)^(true|false)$"`
	DefaultWhitelist         []string    `key:"defaultWhitelist" constraint:"^((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\\/([0-9]|[1-2][0-9]|3[0-2]))?(\\s*,\\s*)?)+$"`
	WhitelistMode            string      `key:"whitelistMode" constraint:"^(extend|override)$"`
	SSLConfig                *SSLConfig  `key:"ssl"`
	AppConfigs               []*AppConfig
	BuilderConfig            *BuilderConfig
	PlatformCertificate      *Certificate
}

func newRouterConfig() *RouterConfig {
	return &RouterConfig{
		WorkerProcesses:          "auto",
		MaxWorkerConnections:     "768",
		TrafficStatusZoneSize:    "1m",
		DefaultTimeout:           "1300s",
		ServerNameHashMaxSize:    "512",
		ServerNameHashBucketSize: "64",
		GzipConfig:               newGzipConfig(),
		BodySize:                 "1m",
		ProxyRealIPCIDRs:         []string{"10.0.0.0/8"},
		ErrorLogLevel:            "error",
		UseProxyProtocol:         false,
		EnforceWhitelists:        false,
		WhitelistMode:            "extend",
		SSLConfig:                newSSLConfig(),
	}
}

// GzipConfig encapsulates gzip configuration.
type GzipConfig struct {
	Enabled     bool   `key:"enabled" constraint:"(?i)^(true|false)$"`
	CompLevel   string `key:"compLevel" constraint:"^[1-9]$"`
	Disable     string `key:"disable"`
	HTTPVersion string `key:"httpVersion" constraint:"^(1\\.0|1\\.1)$"`
	MinLength   string `key:"minLength" constraint:"^\\d+$"`
	Proxied     string `key:"proxied" constraint:"^((off|expired|no-cache|no-store|private|no_last_modified|no_etag|auth|any)\\s*)+$"`
	Types       string `key:"types" constraint:"(?i)^([a-z\\d]+/[a-z\\d][a-z\\d+\\-\\.]*[a-z\\d]\\s*)+$"`
	Vary        string `key:"vary" constraint:"^(on|off)$"`
}

func newGzipConfig() *GzipConfig {
	return &GzipConfig{
		Enabled:     true,
		CompLevel:   "5",
		Disable:     "msie6",
		HTTPVersion: "1.1",
		MinLength:   "256",
		Proxied:     "any",
		Types:       "application/atom+xml application/javascript application/json application/rss+xml application/vnd.ms-fontobject application/x-font-ttf application/x-web-app-manifest+json application/xhtml+xml application/xml font/opentype image/svg+xml image/x-icon text/css text/plain text/x-component",
		Vary:        "on",
	}
}

// AppConfig encapsulates the configuration for all routes to a single back end.
type AppConfig struct {
	Name           string
	Domains        []string `key:"domains" constraint:"(?i)^((([a-z0-9]+(-[a-z0-9]+)*)|((\\*\\.)?[a-z0-9]+(-[a-z0-9]+)*\\.)+[a-z]{2,})(\\s*,\\s*)?)+$"`
	Whitelist      []string `key:"whitelist" constraint:"^((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\\/([0-9]|[1-2][0-9]|3[0-2]))?(\\s*,\\s*)?)+$"`
	ConnectTimeout string   `key:"connectTimeout" constraint:"^[1-9]\\d*(ms|[smhdwMy])?$"`
	TCPTimeout     string   `key:"tcpTimeout" constraint:"^[1-9]\\d*(ms|[smhdwMy])?$"`
	ServiceIP      string
	// Certificate and private key for app domains.
	AppCerts        map[string]*Certificate
	AppCertMappings map[string]string `key:"appCertificates" constraint:"(?i)^((([a-z0-9]+(-[a-z0-9]+)*)|((\\*\\.)?[a-z0-9]+(-[a-z0-9]+)*\\.)+[a-z]{2,}):([a-z0-9]+(-[a-z0-9]+)*)(\\s*,\\s*)?)+$"`
	// If present, open client certificate for app domains.
	ClientCerts        map[string]*Certificate
	ClientCertMappings map[string]string `key:"clientCertificates" constraint:"(?i)^((([a-z0-9]+(-[a-z0-9]+)*)|((\\*\\.)?[a-z0-9]+(-[a-z0-9]+)*\\.)+[a-z]{2,}):([a-z0-9]+(-[a-z0-9]+)*)(\\s*,\\s*)?)+$"`
	EnforceHTTPS       bool              `key:"enforceHTTPS" constraint:"(?i)^(true|false)$"`
	Available          bool
}

func newAppConfig(routerConfig *RouterConfig) *AppConfig {
	return &AppConfig{
		ConnectTimeout: "30s",
		EnforceHTTPS:   false,
		TCPTimeout:     routerConfig.DefaultTimeout,
		AppCerts:       make(map[string]*Certificate, 0),
		ClientCerts:    make(map[string]*Certificate, 0),
	}
}

// BuilderConfig encapsulates the configuration of the deis-builder-- if it's in use.
type BuilderConfig struct {
	ConnectTimeout string `key:"connectTimeout" constraint:"^[1-9]\\d*(ms|[smhdwMy])?$"`
	TCPTimeout     string `key:"tcpTimeout" constraint:"^[1-9]\\d*(ms|[smhdwMy])?$"`
	ServiceIP      string
}

func newBuilderConfig() *BuilderConfig {
	return &BuilderConfig{
		ConnectTimeout: "10s",
		TCPTimeout:     "1200s",
	}
}

// Certificate represents an SSL certificate for use in securing routable applications.
type Certificate struct {
	Cert string
	Key  string
}

func newCertificate(cert string, key string) *Certificate {
	return &Certificate{
		Cert: cert,
		Key:  key,
	}
}

// SSLConfig represents SSL-related configuration options.
type SSLConfig struct {
	Enforce           bool        `key:"enforce" constraint:"(?i)^(true|false)$"`
	Protocols         string      `key:"protocols" constraint:"^((SSLv2|SSLv3|TLSv1|TLSv1\\.1|TLSv1\\.2)\\s*)+$"`
	Ciphers           string      `key:"ciphers" constraint:"^([A-Z][A-Z\\d-]+:?)*$"`
	SessionCache      string      `key:"sessionCache" constraint:"^(off|none|((builtin(:[1-9]\\d*)?|shared:\\w+:[1-9]\\d*[kKmM]?)\\s*){1,2})$"`
	SessionTimeout    string      `key:"sessionTimeout" constraint:"^[1-9]\\d*(ms|[smhdwMy])?$"`
	UseSessionTickets bool        `key:"useSessionTickets" constraint:"(?i)^(true|false)$"`
	BufferSize        string      `key:"bufferSize" constraint:"^[1-9]\\d*[kKmM]?$"`
	HSTSConfig        *HSTSConfig `key:"hsts"`
	DHParam           string
}

func newSSLConfig() *SSLConfig {
	return &SSLConfig{
		Enforce:           false,
		Protocols:         "TLSv1 TLSv1.1 TLSv1.2",
		SessionTimeout:    "10m",
		UseSessionTickets: true,
		BufferSize:        "4k",
		HSTSConfig:        newHSTSConfig(),
	}
}

// HSTSConfig represents configuration options having to do with HTTP Strict Transport Security.
type HSTSConfig struct {
	Enabled           bool `key:"enabled" constraint:"(?i)^(true|false)$"`
	MaxAge            int  `key:"maxAge" constraint:"^[1-9]\\d*$"`
	IncludeSubDomains bool `key:"includeSubDomains" constraint:"(?i)^(true|false)$"`
	Preload           bool `key:"preload" constraint:"(?i)^(true|false)$"`
}

func newHSTSConfig() *HSTSConfig {
	return &HSTSConfig{
		Enabled:           false,
		MaxAge:            15552000, // 180 days
		IncludeSubDomains: false,
		Preload:           false,
	}
}

// Build creates a RouterConfig configuration object by querying the k8s API for
// relevant metadata concerning itself and all routable services.
func Build(kubeClient *client.Client) (*RouterConfig, error) {
	// Get all relevant information from k8s:
	//   deis-router rc or daemonset
	//   All services with label "routable=true"
	//   deis-builder service, if it exists
	// These are used to construct a model...
	routerMeta, err := getRouterMeta(kubeClient)
	if err != nil {
		return nil, err
	}
	appServices, err := getAppServices(kubeClient)
	if err != nil {
		return nil, err
	}
	// builderService might be nil if it's not found and that's ok.
	builderService, err := getBuilderService(kubeClient)
	if err != nil {
		return nil, err
	}
	platformCertSecret, err := getSecret(kubeClient, "deis-router-platform-cert", namespace)
	if err != nil {
		return nil, err
	}
	dhParamSecret, err := getSecret(kubeClient, "deis-router-dhparam", namespace)
	if err != nil {
		return nil, err
	}
	// Build the model...
	routerConfig, err := build(kubeClient, routerMeta, platformCertSecret, dhParamSecret, appServices, builderService)
	if err != nil {
		return nil, err
	}
	return routerConfig, nil
}

func getRouterMeta(kubeClient *client.Client) (*api.ObjectMeta, error) {
	rcClient := kubeClient.ReplicationControllers(namespace)
	rc, err := rcClient.Get("deis-router")

	if err == nil {
		return &rc.ObjectMeta, nil
	}

	dsClient := kubeClient.ExtensionsClient.DaemonSets(namespace)
	ds, err := dsClient.Get("deis-router")

	if err == nil {
		return &ds.ObjectMeta, nil
	}

	log.Printf("Get router meta error: %s", err)
	// TODO: Support Deployment
	return nil, err
}

func getAppServices(kubeClient *client.Client) (*api.ServiceList, error) {
	serviceClient := kubeClient.Services(api.NamespaceAll)
	services, err := serviceClient.List(api.ListOptions{LabelSelector: servicesSelector})
	if err != nil {
		return nil, err
	}
	return services, nil
}

// getBuilderService will return the service named "deis-builder" from the same namespace as
// the router, but will return nil (without error) if no such service exists.
func getBuilderService(kubeClient *client.Client) (*api.Service, error) {
	serviceClient := kubeClient.Services(namespace)
	service, err := serviceClient.Get("deis-builder")
	if err != nil {
		statusErr, ok := err.(*errors.StatusError)
		// If the issue is just that no deis-builder was found, that's ok.
		if ok && statusErr.Status().Code == 404 {
			// We'll just return nil instead of a found *api.Service.
			return nil, nil
		}
		return nil, err
	}
	return service, nil
}

func getSecret(kubeClient *client.Client, name string, ns string) (*api.Secret, error) {
	secretClient := kubeClient.Secrets(ns)
	secret, err := secretClient.Get(name)
	if err != nil {
		statusErr, ok := err.(*errors.StatusError)
		// If the issue is just that no such secret was found, that's ok.
		if ok && statusErr.Status().Code == 404 {
			// We'll just return nil instead of a found *api.Secret
			return nil, nil
		}
		return nil, err
	}
	return secret, nil
}

func build(kubeClient *client.Client, routerMeta *api.ObjectMeta, platformCertSecret *api.Secret, dhParamSecret *api.Secret, appServices *api.ServiceList, builderService *api.Service) (*RouterConfig, error) {
	routerConfig, err := buildRouterConfig(routerMeta, platformCertSecret, dhParamSecret)
	if err != nil {
		return nil, err
	}
	for _, appService := range appServices.Items {
		appConfig, err := buildAppConfig(kubeClient, appService, routerConfig)
		if err != nil {
			return nil, err
		}
		if appConfig != nil {
			routerConfig.AppConfigs = append(routerConfig.AppConfigs, appConfig)
		}
	}
	if builderService != nil {
		builderConfig, err := buildBuilderConfig(builderService)
		if err != nil {
			return nil, err
		}
		if builderConfig != nil {
			routerConfig.BuilderConfig = builderConfig
		}
	}
	return routerConfig, nil
}

func buildRouterConfig(meta *api.ObjectMeta, platformCertSecret *api.Secret, dhParamSecret *api.Secret) (*RouterConfig, error) {
	routerConfig := newRouterConfig()
	err := modeler.MapToModel(meta.Annotations, "nginx", routerConfig)
	if err != nil {
		return nil, err
	}
	if platformCertSecret != nil {
		platformCertificate, err := buildCertificate(platformCertSecret, "platform")
		if err != nil {
			return nil, err
		}
		routerConfig.PlatformCertificate = platformCertificate
	}
	if dhParamSecret != nil {
		dhParam, err := buildDHParam(dhParamSecret)
		if err != nil {
			return nil, err
		}
		routerConfig.SSLConfig.DHParam = dhParam
	}
	return routerConfig, nil
}

func buildAppConfig(kubeClient *client.Client, service api.Service, routerConfig *RouterConfig) (*AppConfig, error) {
	appConfig := newAppConfig(routerConfig)
	appConfig.Name = service.Labels["app"]
	// If we didn't get the app name from the app label, fall back to inferring the app name from
	// the service's own name.
	if appConfig.Name == "" {
		appConfig.Name = service.Name
	}
	// if app name and Namespace are not same then combine the two as it
	// makes deis services (as an example) clearer, such as deis/controller
	if appConfig.Name != service.Namespace {
		appConfig.Name = service.Namespace + "/" + appConfig.Name
	}
	err := modeler.MapToModel(service.Annotations, "", appConfig)
	if err != nil {
		return nil, err
	}
	// If no domains are found, we don't have the information we need to build routes
	// to this application.  Abort.
	if len(appConfig.Domains) == 0 {
		return nil, nil
	}
	// Step through the domains, and decide which cert, if any, will be used for securing each.
	// For each that is a FQDN, we'll look to see if a corresponding cert-bearing secret also
	// exists.  If so, that will be used.  If a domain isn't an FQDN we will use the default cert--
	// even if that is nil.
	for _, domain := range appConfig.Domains {
		if strings.Contains(domain, ".") {
			var certificate *Certificate
			var err error
			// app certificate
			certificate, err = fetchCertificate(kubeClient, appConfig.AppCertMappings, domain, service.Namespace)
			if err != nil {
				return nil, err
			}
			if certificate != nil {
				appConfig.AppCerts[domain] = certificate
			}
			// client certificate
			certificate, err = fetchCertificate(kubeClient, appConfig.ClientCertMappings, domain, service.Namespace)
			if err != nil {
				return nil, err
			}
			if certificate != nil {
				appConfig.ClientCerts[domain] = certificate
			}
		} else {
			// default platform certificate only for app certificate
			appConfig.AppCerts[domain] = routerConfig.PlatformCertificate
		}
	}
	appConfig.ServiceIP = service.Spec.ClusterIP
	endpointsClient := kubeClient.Endpoints(service.Namespace)
	endpoints, err := endpointsClient.Get(service.Name)
	if err != nil {
		return nil, err
	}
	appConfig.Available = len(endpoints.Subsets) > 0 && len(endpoints.Subsets[0].Addresses) > 0
	return appConfig, nil
}

func fetchCertificate(kubeClient *client.Client, mappings map[string]string, domain string, ns string) (*Certificate, error) {
	// Look for a cert-bearing secret for this domain.
	secretName := fmt.Sprintf("%s-cert", mappings[domain])
	certSecret, err := getSecret(kubeClient, secretName, ns)
	if err != nil {
		return nil, err
	}
	if certSecret != nil {
		certificate, err := buildCertificate(certSecret, domain)
		if err != nil {
			return nil, err
		}
		return certificate, nil
	}
	return nil, nil
}

func buildBuilderConfig(service *api.Service) (*BuilderConfig, error) {
	builderConfig := newBuilderConfig()
	builderConfig.ServiceIP = service.Spec.ClusterIP
	err := modeler.MapToModel(service.Annotations, "nginx", builderConfig)
	if err != nil {
		return nil, err
	}
	return builderConfig, nil
}

func buildCertificate(certSecret *api.Secret, context string) (*Certificate, error) {
	cert, ok := certSecret.Data["tls.crt"]
	// If no cert is found in the secret, warn and return nil
	if !ok {
		log.Printf("WARN: The k8s secret intended to convey the %s certificate contained no entry \"tls.crt\".\n", context)
		return nil, nil
	}
	key, ok := certSecret.Data["tls.key"]
	// Changed: Nil private key is acceptable
	if !ok {
		return newCertificate(string(cert[:]), ""), nil
	} else {
		return newCertificate(string(cert[:]), string(key[:])), nil
	}
}

func buildDHParam(dhParamSecret *api.Secret) (string, error) {
	dhParam, ok := dhParamSecret.Data["dhparam"]
	// If no dhparam is found in the secret, warn and return ""
	if !ok {
		log.Println("WARN: The k8s secret intended to convey the dhparam contained no entry \"dhparam\".")
		return "", nil
	}
	return string(dhParam), nil
}
