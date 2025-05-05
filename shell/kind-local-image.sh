# /bin/bash
set -e

# Check if image name is provided
if [ $# -eq 0 ]; then
      echo "Error: Please provide an image name"
      echo "Usage: $0 <image-name>"
      exit 1
fi

IMAGE_PATTERN=$1
# Find matching images
MATCHING_IMAGES=$(docker images --format "{{.Repository}}:{{.Tag}}" | grep -i "$IMAGE_PATTERN" || echo "")

if [ -z "$MATCHING_IMAGES" ]; then
      echo "Error: No images found matching pattern '$IMAGE_PATTERN'"
      exit 1
fi

# If multiple images found, let user choose
if [ $(echo "$MATCHING_IMAGES" | wc -l) -gt 1 ]; then
      echo "Multiple images found:"
      i=1
      while IFS= read -r image; do
            echo "$i) $image"
            i=$((i + 1))
      done <<<"$MATCHING_IMAGES"

      read -p "Please select an image (1-$((i - 1))): " selection

      if ! [[ "$selection" =~ ^[0-9]+$ ]] || [ "$selection" -lt 1 ] || [ "$selection" -gt $((i - 1)) ]; then
            echo "Invalid selection"
            exit 1
      fi

      IMAGE_NAME=$(echo "$MATCHING_IMAGES" | sed -n "${selection}p")
else
      IMAGE_NAME=$MATCHING_IMAGES
fi

# Get all kind nodes
NODES=$(kind get nodes 2>/dev/null || echo "")
if [ -z "$NODES" ]; then
      echo "Error: No kind nodes found. Is your kind cluster running?"
      exit 1
fi

# Load the image to each node
echo "Loading image $IMAGE_NAME to kind nodes..."
for node in $NODES; do
      echo "Loading to node: $node"
      kind load docker-image "$IMAGE_NAME" --nodes "$node"
done

echo "Image successfully loaded to all nodes"
