package main

import (
	"context"
	"flag"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
	"os"
	"path/filepath"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		fmt.Println(h)
		return h
	}
	fmt.Println(os.Getenv("USERPROFILE") )
	return os.Getenv("USERPROFILE") // windows
}

func int32Ptr(i int32) *int32 { return &i }

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file where no home declared")
	}
	flag.Parse()


	// BuildConfigFromFlags is a helper function that builds configs from a master url
	//  or  with a kubeconfig filepath. These are passed in as command line flags for cluster
	// components. Warnings should reflect this usage.
	//If neither masterUrl or kubeconfig Path  are passed in we fallback to inClusterConfig.
	//If inClusterConfig fails, we fallback to the default config.

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	//default config client for every  group
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	 listDeployment(clientset)

	//update deployment

	updateDeployment(clientset)
	listDeployment(clientset)

	////delete deployment

	//deleteDeployment(clientset)
	//listDeployment(clientset)

	//create deployment

	//createDeployment(clientset)

	//create servie
	//createService(clientset)

}
func createDeployment(clientset *kubernetes.Clientset)  {
	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)


	//create a deployment

	deployment := &appsv1.Deployment{
		ObjectMeta : metav1.ObjectMeta{
			Name: "book-server-deployment",

		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app" : "book-api",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app" : "book-api",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name: "book-api-pods",
							Image: "hremon331046/library-management-api:1.1",
							Ports: []apiv1.ContainerPort{
								{
									Name: "http",
									Protocol: apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	// deploy the deployment creation via deploymentsClient.create and save the final onject in result

	result , err := deploymentsClient.Create(context.TODO(),deployment, metav1.CreateOptions{})


	if err != nil {
		panic(err)
	}
	fmt.Println(result.Name," pods ", *result.Spec.Replicas)


}
func updateDeployment(clientset *kubernetes.Clientset) {
	deploymentClientset := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)
	// Modify the "result" returned by Get() and retry Update(result) until
	//	you no longer get a conflict error. This way, you can preserve changes made
	//	by other clients between Create() and Update(). This is implemented below
	//	using the retry utility package included with client-go.
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {

		//first retrive the latest version of the desired deplooyment
		result , getErr := deploymentClientset.Get(context.TODO(),"book-server-deployment",metav1.GetOptions{})

		if getErr != nil {
			panic(fmt.Errorf("Failed to get the latest version of deployment : %v ",getErr))

		}
		//change the number of replicas
		result.Spec.Replicas = int32Ptr(3)
		//replaced hremon331046/library-management-api:1.1 with hremon331046/library-management-api:1.0

		result.Spec.Template.Spec.Containers[0].Image = "hremon331046/library-management-api:1.0"
		_, updateErr := deploymentClientset.Update(context.TODO(),result,metav1.UpdateOptions{})
		return updateErr
	})
	if retryErr != nil {
		panic(fmt.Errorf("update failed. Error : %v",retryErr))
	}
}

func deleteDeployment(clientset *kubernetes.Clientset) {
	deploymentClientset := clientset.AppsV1().Deployments(metav1.NamespaceDefault)

	deletePolicy := metav1.DeletePropagationForeground
	deleteErr := deploymentClientset.Delete(context.TODO(),"book-server-deployment",metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if deleteErr != nil {
		panic(fmt.Errorf("error when trying to delete . error ; %v",deleteErr))
	}
}
func listDeployment(clientset *kubernetes.Clientset){
	deploymentClient := clientset.AppsV1().Deployments(metav1.NamespaceDefault)

	deployments, err :=deploymentClient.List(context.TODO(),metav1.ListOptions{})
	if err != nil {
		panic(fmt.Errorf("error in listing the deployment %v ",err))

	}
	for _, dep := range deployments.Items {
		fmt.Println("deployment name :  ",dep.Name," replicas : ", *dep.Spec.Replicas)

	}
}

func createService(clientset *kubernetes.Clientset){
	svcClient := clientset.CoreV1().Services(metav1.NamespaceDefault)

	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "book-server-service",

		},
		Spec: apiv1.ServiceSpec{
			Type: apiv1.ServiceTypeNodePort,
			Selector: map[string]string{
				"api": "book-server",
			},
			Ports: []apiv1.ServicePort{
				{
					TargetPort: intstr.IntOrString{
						IntVal: 8080,
					},
					Port: 80,
					NodePort: 30007,
				},
			},
		},

	}
	result , err := svcClient.Create(context.TODO(),svc,metav1.CreateOptions{})
	if err != nil {
		panic(fmt.Errorf("service creation failed . Error :",err))

	}
	fmt.Println("created service ", result.GetObjectMeta().GetName())

}