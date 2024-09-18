document.addEventListener('DOMContentLoaded', async function () {
    const namespace = "sman"; // Default namespace
    const secretList = document.getElementById('secretList');
    
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
              <span class="edit">✏️</span>
              <span class="delete">🗑️</span>
            </td>`;
          secretList.appendChild(secretItem);
        });
      } catch (error) {
        secretList.textContent = `Error: ${error.message}`;
      }
    }
  
    // Load secrets initially
    loadSecrets();
  
    // Modal logic
    const modal = document.getElementById("secretModal");
    const createNewSecretBtn = document.getElementById("createNewSecretBtn");
    const closeModalBtn = document.querySelector(".close");
  
    // Show modal when button is clicked
    createNewSecretBtn.onclick = function() {
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
  
    // Handle secret creation form submission
    document.getElementById('secretForm').addEventListener('submit', async function (event) {
      event.preventDefault();
  
      const secretName = document.getElementById('secretName').value;
      const dataKey = document.getElementById('dataKey').value;
      const dataValue = document.getElementById('dataValue').value;
      const statusMessage = document.getElementById('statusMessage');
  
      // Prepare the payload
      const secretData = {
        name: secretName,
        namespace: namespace,
        data: {
          [dataKey]: dataValue
        }
      };
  
      try {
        const response = await fetch('/secrets', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(secretData),
        });
  
        if (response.ok) {
          const responseData = await response.text();
          statusMessage.textContent = `Success: ${responseData}`;
          statusMessage.style.color = 'green';
  
          // Reload the secret list
          loadSecrets();
  
          // Hide the modal after success
          modal.style.display = "none";
        } else {
          const errorMessage = await response.text();
          statusMessage.textContent = `Error: ${errorMessage}`;
          statusMessage.style.color = 'red';
        }
      } catch (error) {
        statusMessage.textContent = `Error: ${error.message}`;
        statusMessage.style.color = 'red';
      }
    });
  });
  