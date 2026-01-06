#!/bin/bash

# Script to run migrations on the development database
echo "Running migrations on development database..."

# Get access token
ACCESS_TOKEN=$(gcloud auth print-access-token)

# Read the migration files and execute them
for migration_file in migrations/*.sql; do
    echo "Executing migration: $migration_file"
    
    # Read the SQL content
    sql_content=$(cat "$migration_file")
    
    # Create the request body
    request_body=$(cat <<EOF
{
  "sql": "$(echo "$sql_content" | sed 's/"/\\"/g' | tr '\n' ' ')"
}
EOF
)
    
    # Execute the SQL using Cloud SQL Admin API
    response=$(curl -s -X POST \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$request_body" \
        "https://sqladmin.googleapis.com/sql/v1beta4/projects/carpooly-login/instances/carpool-dev1/databases/carpool_dev/execute")
    
    if [[ $response == *"error"* ]]; then
        echo "❌ Failed to execute: $migration_file"
        echo "Response: $response"
        exit 1
    else
        echo "✅ Successfully executed: $migration_file"
    fi
done

echo "🎉 All migrations completed successfully!" 