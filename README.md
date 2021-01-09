# validate-requested-sources

Step1. Run ./build.sh to build the docker image
Step2. push the docker image 
Step3. Create private and public keys using which a CSR needs to be created and approved
Step4. Use the base64 encoded key in caBundle for webhook.
Step5. use kubectl create -k in dev kustomize folder
Step6. trying to create a deployment that exceeds the resource limit from testing folder should show an error
