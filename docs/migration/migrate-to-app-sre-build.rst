Migrate to App-SRE Build Pipeline
=================================

# Copy deployment template from saas-templates into source code repo
# Add build_deploy.sh and pr_check.sh to source code repo
# Ensure code repo has a Dockerfile that does not pull form Dockerhub
# Create PR check and build_master jenkins jobs in app-interface
# Modify saas-deploy file for service
