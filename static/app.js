document.addEventListener('DOMContentLoaded', async function () {
  const namespace = "sman"; // Default namespace
  const secretList = document.getElementById('secretList');
  const statusMessage = document.getElementById('statusMessage');
  
  // Utility to display log messages
  function showMessage(message, type = 'success') {
    statusMessage.textContent = message;
    statusMessage.className = type;
    statusMessage.style.visibility = 'visible';
  }

  // Fetch secrets on page load
  async function loadSecrets() {
    try {
      const response = await fetch(`/secrets?namespace=${namespace}`);
      const secrets = await response.json();
      
      secretList.innerHTML = '';
      
      secrets.forEach(secret => {
        let secretKeyValue = "";

        // Loop over all key-value pairs in secret.data
        for (let key in secret.data) {
          if (secret.data.hasOwnProperty(key)) {
            secretKeyValue += `${key}: ${secret.data[key]}, `;
          }
        }

        // Remove the trailing comma and space
        secretKeyValue = secretKeyValue.slice(0, -2);

        const secretItem = document.createElement('tr');
        secretItem.innerHTML = `
          <td>${secret.name}</td>
          <td>${secretKeyValue || "No data"}</td>
          <td class="actions">
            <span class="edit" data-name="${secret.name}" data-key="${Object.keys(secret.data)[0]}" data-value="${Object.values(secret.data)[0]}">✏️</span>
            <span class="delete" data-name="${secret.name}">🗑️</span>
          </td>`;
        secretList.appendChild(secretItem);
      });

      // Add event listeners for the edit and delete buttons
      addActionEventListeners();
    } catch (error) {
      showMessage(`Error: ${error.message}`, 'error');
    }
  }

  // Load secrets initially
  loadSecrets();

  // Modal logic
  const modal = document.getElementById("secretModal");
  const createNewSecretBtn = document.getElementById("createNewSecretBtn");
  const closeModalBtn = document.querySelector(".close");
  const actionType = document.getElementById("actionType");

  // Show modal when button is clicked for creating
  createNewSecretBtn.onclick = function() {
    document.getElementById('modalTitle').textContent = 'Create New Secret';
    document.getElementById('secretForm').reset(); // Reset the form
    actionType.value = 'create'; // Set the action type to create
    modal.style.display = "block";
  }

  // Close the modal when the close button is clicked
  closeModalBtn.onclick = function() {
    modal.style.display = "none";
  }

  // Close the modal if clicked outside of the content
  window.onclick = function(event) {
    if (event.target == modal) {
      modal.style.display = "none";
    }
  }

  // Handle secret creation and updating form submission
  document.getElementById('secretForm').addEventListener('submit', async function (event) {
    event.preventDefault();

    const secretName = document.getElementById('secretName').value;
    const dataKey = document.getElementById('dataKey').value;
    const dataValue = document.getElementById('dataValue').value;

    // Prepare the payload
    const secretData = {
      name: secretName,
      namespace: namespace,
      data: {
        [dataKey]: dataValue
      }
    };

    const method = actionType.value === 'edit' ? 'PUT' : 'POST'; // Determine whether to create or update
    const url = '/secrets';

    try {
      const response = await fetch(url, {
        method: method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(secretData),
      });

      if (response.ok) {
        const responseData = await response.text();
        showMessage(`Success: ${responseData}`, 'success');

        // Reload the secret list
        loadSecrets();

        // Hide the modal after success
        modal.style.display = "none";
      } else {
        const errorMessage = await response.text();
        showMessage(`Error: ${errorMessage}`, 'error');
      }
    } catch (error) {
      showMessage(`Error: ${error.message}`, 'error');
    }
  });

  // Function to add event listeners for the edit and delete buttons
  function addActionEventListeners() {
    // Add event listener for edit buttons
    document.querySelectorAll('.edit').forEach(btn => {
      btn.addEventListener('click', function () {
        const secretName = btn.getAttribute('data-name');
        const dataKey = btn.getAttribute('data-key');
        const dataValue = btn.getAttribute('data-value');

        document.getElementById('secretName').value = secretName;
        document.getElementById('dataKey').value = dataKey;
        document.getElementById('dataValue').value = dataValue;

        document.getElementById('modalTitle').textContent = 'Edit Secret';
        actionType.value = 'edit'; // Set the action type to edit

        modal.style.display = "block";
      });
    });

    // Add event listener for delete buttons
    document.querySelectorAll('.delete').forEach(btn => {
      btn.addEventListener('click', async function () {
        const secretName = btn.getAttribute('data-name');

        try {
          const response = await fetch(`/secrets?namespace=${namespace}&name=${secretName}`, {
            method: 'DELETE',
          });

          if (response.ok) {
            const responseData = await response.text();
            showMessage(`Success: ${responseData}`, 'success');

            // Reload the secret list
            loadSecrets();
          } else {
            const errorMessage = await response.text();
            showMessage(`Error: ${errorMessage}`, 'error');
          }
        } catch (error) {
          showMessage(`Error: ${error.message}`, 'error');
        }
      });
    });
  }
});
