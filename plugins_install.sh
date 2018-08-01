#!/bin/bash

# This script will install and uninstall the desired plugins needed to run 
# genie test suite
#plugins can be independantly installed as well using differnt options

desiredState="Running"
maxWaitSeconds=100   # Set interval (duration) in seconds.
elapsedSeconds=0   # updated after every cycle of plugin state verification
isFlannelUp=false 
isWeaveUp=false
isCalicoUp=false
isRomanaUp=false

Install_Flannel() {
  status=(`kubectl get pod --all-namespaces | grep -E "kube-flannel" | awk '{print $4}'`)
  if [ "$status" == "$desiredState" ]; then
    echo "Flannel already running"
  else
    kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/v0.9.1/Documentation/kube-flannel.yml
    echo "Flannel is started"
  fi

}

Install_Weave() {
  status=(`kubectl get pod --all-namespaces | grep -E "weave-net" | awk '{print $4}'`)
  if [ "$status" == "$desiredState" ]; then
    echo "Weave already running"
  else
    kubectl apply -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')"
    echo "Weave is started"
  fi
}

Install_Romana() {
  status=(`kubectl get pod --all-namespaces | grep -E "romana-agent" | awk '{print $4}'`)
  if [ "$status" == "$desiredState" ]; then
    echo "Romana already running"
  else
    kubectl apply -f https://raw.githubusercontent.com/romana/romana/master/containerize/specs/romana-kubeadm.yml
    echo "Romana is started"
  fi
}

Install_Calico() {
  status=(`kubectl get pod --all-namespaces | grep -E "calico" | awk '{print $4}'`)
  if [ "$status" == "$desiredState" ]; then
    echo "Calico already running"
  else
    kubectl apply -f https://docs.projectcalico.org/v3.0/getting-started/kubernetes/installation/hosted/kubeadm/1.7/calico.yaml
    echo "Calico is started"
  fi
}

# CAdvisor will be used to get network usage statistics to support smart plugin selection
Install_CAdvisor() {
  docker rm $(docker ps -q -f status=exited)
  sudo docker run --volume=/:/rootfs:ro --volume=/var/run:/var/run:rw --volume=/sys:/sys:ro --volume=/var/lib/docker/:/var/lib/docker:ro --publish=4194:4194 --detach=true --name=cadvisor google/cadvisor:latest --logtostderr --port=4194
}

Delete_AllPlugins() {
  kubectl delete -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')"
  kubectl delete -f https://raw.githubusercontent.com/coreos/flannel/v0.9.1/Documentation/kube-flannel.yml
  kubectl delete -f https://raw.githubusercontent.com/romana/romana/master/containerize/specs/romana-kubeadm.yml
  kubectl delete -f https://docs.projectcalico.org/v2.6/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml
  echo "All plugins deleted"
}


Plugins_CNI()
{
  Install_Flannel
  Install_Weave
  Install_Romana
  Install_Calico
  Install_CAdvisor
  while [ $elapsedSeconds -lt $maxWaitSeconds ]
  do
    if [ $isFlannelUp == false ]; then
      Flannel_status=(`kubectl get pod --all-namespaces | grep -E "kube-flannel" | awk '{print $4}'`)
      if [ "$Flannel_status" == "$desiredState" ] ; then
        echo "Flannel is running"
        isFlannelUp=true 
      fi
    fi

    if [ $isWeaveUp == false ]; then
      Weave_status=(`kubectl get pod --all-namespaces | grep -E "weave-net" | awk '{print $4}'`)
      if [ "$Weave_status" == "$desiredState" ]; then
        echo "Weave is running"
        isWeaveUp=true
      fi
    fi

    if [ $isRomanaUp == false ];then
      Romana_status=(`kubectl get pod --all-namespaces | grep -E "romana-agent" | awk '{print $4}'`)
      if [ "$Romana_status" == "$desiredState" ]; then
        echo "Romana is running"
        isRomanaUp=true
      fi
    fi

    if [ $isCalicoUp == false ];then
      Calico_status=(`kubectl get pod --all-namespaces | grep -E "calico" | awk '{print $4}'`)
      if [ "$Calico_status" == "$desiredState" ]; then
        echo "Calico is running"
        isCalicoUp=true
      fi
    fi

    if [ $isFlannelUp == true ] && [ $isWeaveUp == true ] && [ $isRomanaUp == true ] && [ $isCalicoUp == true ]; then
      echo "All desired plugins came to running state"
      break;
    fi
    elapsedSeconds=`expr $elapsedSeconds + 1`
  done
}
options () {
  echo "please provide Valid option"

  echo  "valid options are......."

  echo  "1.-all---to install all plugin only"
  echo  "2.-flannel ----to install flannel only"
  echo  "3.-weave---to install Weave only"
  echo  "4.-calico----to install Calico only"
  echo  "5.-romana---To install romana only"
  echo  "6.-deleteall---To delete all"
}


run () {
  echo $@
  echo $1
  flag=0
  declare -a input=("-all" "-flannel" "-weave" "-romana" "-calico" "-deleteall")
  for i in "${input[@]}"
  do
    if [ "$i" == "$1" ]; then
      flag=1
    fi
  done


  if [ $flag -eq 0 ]; then
    options
    exit 1
  fi

  case $1 in
    "-all" ) Plugins_CNI;;
    "-flannel" ) Install_Flannel ;;
    "-weave" ) Install_Weave ;;
    "-romana" ) Install_Romana ;;
    "-calico" ) Install_Calico ;;
    "-deleteall" ) Delete_AllPlugins ;;
    *)
    ;;
  esac
}

run $@

